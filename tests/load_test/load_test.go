package load_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/prometheus/common/model"
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
	ctx := context.Background()
	t := tests.GetT()

	var overwatch promquery.PrometheusClient

	BeforeAll(func() {
		var err error
		overwatch, err = tests.SetupOverwatchClient(ctx, t)
		require.NoError(t, err)

		install.InstallVMGather(t)
		install.InstallVMK8StackWithHelm(
			ctx,
			consts.VMK8sStackChart,
			consts.SmokeValuesFile,
			t,
			consts.DefaultVMNamespace,
			consts.DefaultReleaseName,
		)
		install.InstallOverwatch(ctx, t, consts.OverwatchNamespace, consts.DefaultVMNamespace, consts.DefaultReleaseName)

		// Install k6 operator
		install.InstallK6(ctx, t, consts.K6OperatorNamespace)

		// Build prometheus benchmark configuration
		vmSelectSvcAddr := consts.GetVMSelectSvc(consts.DefaultReleaseName, consts.DefaultVMNamespace)
		vmInsertSvcAddr := consts.GetVMInsertSvc(consts.DefaultReleaseName, consts.DefaultVMNamespace)

		prombenchConfig := tests.PromBenchmarkConfig{
			DisableMonitoring: true,
			TargetsCount:      "500",
			WriteURL:          fmt.Sprintf("http://%s/insert/0/prometheus/api/v1/write", vmInsertSvcAddr),
			ReadURL:           fmt.Sprintf("http://%s/select/0/prometheus", vmSelectSvcAddr),
		}
		install.InstallPrometheusBenchmark(ctx, t, consts.BenchmarkNamespace, prombenchConfig.ToHelmValues())

		// Prepare namespace for k6 tests
		kubeOpts := k8s.NewKubectlOptions("", "", consts.K6TestsNamespace)
		k8s.CreateNamespace(t, kubeOpts, consts.K6TestsNamespace)
	})

	AfterEach(func() {
		defer func() {
			kubeOpts := k8s.NewKubectlOptions("", "", consts.K6TestsNamespace)
			k8s.DeleteNamespace(t, kubeOpts, consts.K6TestsNamespace)
		}()

		gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, consts.DefaultReleaseName)
	})

	Describe("Inner", func() {
		It("Default installation should handle 50vus-30mins load test scenario", Label("kind", "gke", "id=d37b1987-a9e7-4d13-87b7-f2ded679c249"), func() {
			By("Run 50vus-30mins scenario")
			scenario := "vmselect-50vus-30mins"

			// Build VMSelect URL using constants
			vmSelectSvcAddr := consts.GetVMSelectSvc(consts.DefaultReleaseName, consts.DefaultVMNamespace)
			vmSelectURL := fmt.Sprintf("http://%s/select/0/prometheus/api/v1/query_range", vmSelectSvcAddr)

			err := install.RunK6Scenario(ctx, t, consts.K6TestsNamespace, consts.DefaultVMNamespace, scenario, vmSelectURL, 3)
			require.NoError(t, err)

			By("Waiting for K6 jobs to complete")
			install.WaitForK6JobsToComplete(ctx, t, consts.K6TestsNamespace, scenario, 3)

			By("No alerts are firing")
			overwatch.CheckNoAlertsFiring(ctx, t, consts.DefaultVMNamespace, nil)

			By("At least 50m rows were inserted")
			value, err := overwatch.VectorValue(ctx, "sum (vm_rows_inserted_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(40_000_000))

			By("At least 400k merges were made")
			value, err = overwatch.VectorValue(ctx, "sum(vm_rows_merged_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(400_000))

			By("No rows were ignored")
			value, err = overwatch.VectorValue(ctx, "sum (vm_rows_ignored_total)")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(0))

			value, err = overwatch.VectorValue(ctx, "sum (vm_rows_invalid_total)")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(0))

			By("At least 100k requests were made")
			value, err = overwatch.VectorValue(ctx, "sum(vm_requests_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(10_000))
		})
	})
})
