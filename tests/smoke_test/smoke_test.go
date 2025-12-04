package smoke_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestSmokeTests(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Smoke test Suite", suiteConfig, reporterConfig)
}

var _ = Describe("Smoke test", Ordered, Label("smoke"), func() {
	const (
		namespace   = "vm"
		releaseName = "vmks"
		helmChart   = "vm/victoria-metrics-k8s-stack"
		valuesFile  = "../../manifests/smoke.yaml"
	)

	ctx := context.Background()

	ctxCancel, cancel := context.WithCancel(ctx)
	AfterAll(func() {
		cancel()
	})

	t := tests.GetT()

	overwatch, err := promquery.NewPrometheusClient("http://localhost:8429/prometheus")
	require.NoError(t, err)

	BeforeAll(func() {
		overwatch.Start = time.Now()

		install.InstallWithHelm(ctx, helmChart, valuesFile, t, namespace, releaseName)
	})
	AfterEach(func() {
		gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)
		gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout, namespace)
	})

	It("Default installation should handle select requests for 5 mins", Label("kind", "gke", "id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {

		By("Port-forward vmselect address")
		cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmselect-vmks", "8481:8481")
		go func() {
			stdoutStderr, err := cmd.CombinedOutput()
			logger.Default.Logf(t, "vmselect port-forward output: %s", stdoutStderr)
			logger.Default.Logf(t, "vmselect port-forward err: %v", err)
		}()
		// Hack: give it some time to start
		time.Sleep(10 * time.Second)

		By("Send requests for 5 minutes")
		tickerPeriod := time.Second

		promAPI, err := promquery.NewPrometheusClient("http://localhost:8481/select/0/prometheus")
		promAPI.Start = overwatch.Start
		require.NoError(t, err)

		ticker := time.NewTicker(tickerPeriod)
		defer ticker.Stop()

		started := time.Now()
		for ; true; <-ticker.C {
			_, _, err := promAPI.Query(ctx, "up")
			require.NoError(t, err)

			now := <-ticker.C
			if now.Sub(started) > 5*time.Minute {
				break
			}
		}

		By("Setup port-forwarding for overwatch")
		cmd = exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmsingle-overwatch", "8429:8429")
		go func() {
			stdoutStderr, err := cmd.CombinedOutput()
			logger.Default.Logf(t, "overwatch port-forward output: %s", stdoutStderr)
			logger.Default.Logf(t, "overwatch port-forward err: %v", err)
		}()
		// Hack: give it some time to start
		time.Sleep(10 * time.Second)

		By("No alerts are firing")
		value, err := overwatch.VectorValue(ctx, `sum by (alertname) (vmalert_alerts_firing{alertname!~"(InfoInhibitor|Watchdog)"})`)
		require.NoError(t, err)
		require.Zero(t, value)

		// Expect to make at least 10k requests
		By("At least 10k requests were made")
		value, err = overwatch.VectorValue(ctx, "sum(vm_requests_total)")
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, float64(10000))
	})
})
