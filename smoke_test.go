package smoke_test

import (
	"context"
	"fmt"
	"testing"
	"time"

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

var _ = Describe("Smoke test", Ordered, ContinueOnFailure, Label("smoke"), func() {
	const (
		namespace   = "vm"
		releaseName = "vmks"
		helmChart   = "vm/victoria-metrics-k8s-stack"
		valuesFile  = "../../manifests/smoke.yaml"
	)

	ctx := context.Background()
	t := tests.GetT()

	overwatch, err := promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl))
	require.NoError(t, err)

	BeforeAll(func() {
		overwatch.Start = time.Now()

		install.InstallWithHelm(ctx, helmChart, valuesFile, t, namespace, releaseName)
	})
	AfterEach(func() {
		gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)
		gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout, namespace)
	})

	Describe("Inner", func() {
		It("Default installation should handle select requests for 5 mins", Label("kind", "gke", "id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {

			By("Send requests for 5 minutes")
			tickerPeriod := time.Second

			promAPI, err := promquery.NewPrometheusClient(fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl))
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

			By("No alerts are firing")
			overwatch.CheckNoAlertsFiring(ctx, t, []string{})

			// Expect to make at least 10k requests
			By("At least 10k requests were made")
			value, err := overwatch.VectorValue(ctx, "sum(vm_requests_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(10000))
		})
	})
})
