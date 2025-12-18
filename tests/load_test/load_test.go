package load_test

import (
	"context"
	"fmt"
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
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Load test Suite", suiteConfig, reporterConfig)
}

var _ = Describe("Load tests", Ordered, ContinueOnFailure, Label("load-test"), func() {
	const (
		vmNamespace         = "vm1"
		overwatchNamespace  = "overwatch"
		k6OperatorNamespace = "k6-operator-system"
		k6TestsNamespace    = "k6-tests"
		releaseName         = "vmks"
		helmChart           = "vm/victoria-metrics-k8s-stack"
		valuesFile          = "../../manifests/smoke.yaml"
	)

	ctx := context.Background()

	t := tests.GetT()

	var overwatch promquery.PrometheusClient

	BeforeAll(func() {
		install.DiscoverIngressHost(ctx, t)

		var err error
		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl())
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl()))
		require.NoError(t, err)
		overwatch.Start = time.Now()

		install.InstallVMGather(t)
		install.InstallWithHelm(ctx, helmChart, valuesFile, t, vmNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, vmNamespace, releaseName)

		// Install k6 operator
		install.InstallK6(ctx, t, k6OperatorNamespace)

		// Prepare namespace for k6 tests
		kubeOpts := k8s.NewKubectlOptions("", "", k6TestsNamespace)
		k8s.CreateNamespace(t, kubeOpts, k6TestsNamespace)
	})
	AfterEach(func() {
		defer func() {
			kubeOpts := k8s.NewKubectlOptions("", "", k6TestsNamespace)
			k8s.DeleteNamespace(t, kubeOpts, k6TestsNamespace)
		}()

		gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, vmNamespace)
	})

	Describe("Inner", func() {
		It("Default installation should handle 50vus-30mins load test scenario", Label("kind", "gke", "id=d37b1987-a9e7-4d13-87b7-f2ded679c249"), func() {
			By("Run 50vus-30mins scenario")
			scenario := "vmselect-50vus-30mins"
			err := install.RunK6Scenario(ctx, t, k6TestsNamespace, vmNamespace, scenario, 3)
			require.NoError(t, err)

			By("Waiting for K6 jobs to complete")
			install.WaitForK6JobsToComplete(ctx, t, k6TestsNamespace, scenario, 3)

			By("No alerts are firing")
			overwatch.CheckNoAlertsFiring(ctx, t, vmNamespace, []string{
				// TODO[vrutkovs]: sort out these exceptions? These are probably kind-specific
				"TooManyLogs",
				"RecordingRulesError",
				"AlertingRulesError",
			})

			// Expect to make at least 40k requests
			By("At least 10k requests were made")
			value, err := overwatch.VectorValue(ctx, "sum(vm_requests_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(10000))
		})
	})
})
