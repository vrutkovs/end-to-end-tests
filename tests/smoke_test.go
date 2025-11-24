package end_to_end_tests_test

import (
	"context"
	"os/exec"
	"time"

	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
)

var _ = Describe("Smoke test", Ordered, Label("smoke"), func() {

	Context("k8s-stack", func() {
		const (
			namespace   = "vm"
			releaseName = "vmks"
			valuesFile  = "../manifests/smoke.yaml"
		)

		ctx := context.Background()

		ctxCancel, cancel := context.WithCancel(ctx)
		AfterAll(func() {
			cancel()
		})

		t := GetT()

		overwatch, err := promquery.NewPrometheusClient("http://localhost:8481/select/0/prometheus")
		require.NoError(t, err)

		BeforeAll(func() {
			overwatch.Start = time.Now()
			install.InstallWithHelm(ctx, "vm/victoria-metrics-k8s-stack", valuesFile, t, namespace, releaseName)
		})
		AfterEach(func() {
			gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)
			gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout)
		})

		It("should handle select requests", Label("id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {
			By("port-forward vmselect address")
			cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmselect-vmks", "8481:8481")
			go cmd.Run()
			// Hack: give it some time to start
			time.Sleep(1 * time.Second)

			By("send requests for 5 minutes")
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

			// Expect to make at least 40k requests
			By("at least 10k requests were made")
			value, err := overwatch.VectorValue(ctx, "sum(vm_requests_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(10000))
		})
	})
})
