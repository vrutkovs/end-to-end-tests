package load_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
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

func TestLoadTestsTests(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.LabelFilter = "load-test"
	RunSpecs(t, "Load test Suite", suiteConfig, reporterConfig)
}

var _ = Describe("Load tests", Ordered, Label("load-test"), func() {
	const (
		vmNamespace         = "vm"
		k6OperatorNamespace = "k6-operator-system"
		k6TestsNamespace    = "k6-tests"
		releaseName         = "vmks"
		helmChart           = "vm/victoria-metrics-k8s-stack"
		valuesFile          = "../../manifests/smoke.yaml"
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
		install.InstallWithHelm(ctx, helmChart, valuesFile, t, vmNamespace, releaseName)

		// Install k6 operator
		install.InstallK6(ctx, t, k6OperatorNamespace)

		// Prepare namespace for k6 tests
		kubeOpts := k8s.NewKubectlOptions("", "", k6TestsNamespace)
		k8s.CreateNamespace(t, kubeOpts, k6TestsNamespace)
	})
	AfterEach(func() {
		gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)

		kubeOpts := k8s.NewKubectlOptions("", "", k6TestsNamespace)
		k8s.DeleteNamespace(t, kubeOpts, k6TestsNamespace)

		gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout, vmNamespace)
	})

	It("Default installation should handle 50vus-30mins load test scenario", Label("kind", "id=d37b1987-a9e7-4d13-87b7-f2ded679c249"), func() {
		By("Run 50vus-30mins scenario")
		scenario := "vmselect-50vus-30mins"
		err := install.RunK6Scenario(ctx, t, k6TestsNamespace, scenario, 3)
		require.NoError(t, err)
		install.WaitForK6JobsToComplete(ctx, t, k6TestsNamespace, scenario, 3)

		By("Setup port-forwarding for overwatch")
		cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmsingle-overwatch", "8429:8429")
		go func() {
			stdoutStderr, err := cmd.CombinedOutput()
			logger.Default.Logf(t, "vmselect port-forward output: %s", stdoutStderr)
			logger.Default.Logf(t, "vmselect port-forward err: %v", err)
		}()
		// Hack: give it some time to start
		time.Sleep(1 * time.Second)

		By("No alerts are firing")
		value, err := overwatch.VectorValue(ctx, `sum by (alertname) (vmalert_alerts_firing{alertname!~"(InfoInhibitor|Watchdog|TooManyLogs|RecordingRulesError|AlertingRulesError)"})`)
		require.NoError(t, err)
		require.Zero(t, value)

		// Expect to make at least 40k requests
		By("At least 10k requests were made")
		value, err = overwatch.VectorValue(ctx, "sum(vm_requests_total)")
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, float64(10000))
	})
})
