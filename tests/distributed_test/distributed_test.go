package distributed_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/remotewrite"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestDistributedChartTests(t *testing.T) {
	tests.Init()

	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	// suiteConfig.FocusStrings = []string{"should handle load test"}
	RunSpecs(t, "DistributedChart test Suite", suiteConfig, reporterConfig)
}

var (
	t         terratesting.TestingT
	namespace string
	overwatch promquery.PrometheusClient
	c         *http.Client
)

const (
	releaseName         = "vmks"
	overwatchNamespace  = "overwatch"
	vmNamespace         = "monitoring"
	k8sHelmChart        = "vm/victoria-metrics-k8s-stack"
	k8sValuesFile       = "../../manifests/smoke.yaml"
	vmHelmChart         = "vm/victoria-metrics-distributed"
	vmValuesFile        = "../../manifests/distributed.yaml"
	benchmarkNamespace  = "vm-benchmark"
	k6OperatorNamespace = "k6-operator-system"
	k6TestsNamespace    = "k6-tests"
)

// Install VM from helm chart for the first process, set namespace for the rest
var _ = SynchronizedBeforeSuite(
	func(ctx context.Context) {
		t = tests.GetT()
		install.DiscoverIngressHost(ctx, t)
		install.InstallVMGather(t)
		install.InstallVMK8StackWithHelm(ctx, k8sHelmChart, k8sValuesFile, t, vmNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, vmNamespace, releaseName)
		// Remove stock VMCluster - it would be recreated in vm* namespaces
		kubeOpts := k8s.NewKubectlOptions("", "", vmNamespace)
		install.DeleteVMCluster(t, kubeOpts, releaseName)

		// Prepare namespace for k6 tests
		kubeOpts = k8s.NewKubectlOptions("", "", k6TestsNamespace)
		if _, err := k8s.GetNamespaceE(t, kubeOpts, k6OperatorNamespace); err != nil {
			k8s.CreateNamespace(t, kubeOpts, k6TestsNamespace)
		}

		install.InstallK6(ctx, t, k6OperatorNamespace)
	}, func(ctx context.Context) {
		t = tests.GetT()
		namespace = fmt.Sprintf("vm%d", GinkgoParallelProcess())
	},
)

var _ = Describe("Distributed chart", Label("vmcluster"), func() {
	BeforeEach(func(ctx context.Context) {
		install.DiscoverIngressHost(ctx, t)
		var err error
		c = &http.Client{
			Timeout: time.Second * 10,
		}

		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl())
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl()))
		require.NoError(t, err)
		overwatch.Start = time.Now()
	})

	AfterEach(func(ctx context.Context) {
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)
		helmOpts := &helm.Options{
			KubectlOptions: kubeOpts,
		}
		helm.Delete(t, helmOpts, releaseName, true)
		k8s.RunKubectl(t, kubeOpts, "delete", "namespace", namespace, "--ignore-not-found=true")

		if CurrentSpecReport().Failed() {
			gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, releaseName)
			gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		}
	})

	It("should support reading and writing over global and local endpoints", Label("gke", "id=b81bf219-e97c-49fc-8050-8d80153224c7"), func(ctx context.Context) {
		By(fmt.Sprintf("Installing distributed-chart in namespace %s", namespace))
		install.InstallVMDistributedWithHelm(ctx, vmHelmChart, vmValuesFile, t, namespace, releaseName)

		By("Insert data into global write endpoint")
		globalInsertURL := fmt.Sprintf("http://%s/api/v1/write", consts.VMInsertHost(namespace))
		ts := remotewrite.GenTimeSeries("foo", 10, 1)
		err := remotewrite.RemoteWrite(c, ts, globalInsertURL)
		require.NoError(t, err)

		time.Sleep(30 * time.Second)

		By("No alerts are firing")
		// overwatch.CheckNoAlertsFiring(ctx, t, vmNamespace, nil)

		By("Read data from global read endpoint")
		globalSelectURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(namespace))
		globalProm, err := promquery.NewPrometheusClient(globalSelectURL)
		require.NoError(t, err)
		globalProm.Start = overwatch.Start

		value, err := globalProm.VectorValue(ctx, "foo_2")
		require.NoError(t, err)
		require.Equal(t, value, model.SampleValue(1))

		for _, zone := range []string{"europe-central2-a", "europe-central2-b", "europe-central2-c"} {
			By(fmt.Sprintf("Read data from zone %s endpoint", zone))
			zoneSelectURL := fmt.Sprintf("http://vmselect-%s.%s.nip.io/select/0/prometheus", zone, consts.NginxHost())
			tenantProm, err := promquery.NewPrometheusClient(zoneSelectURL)
			require.NoError(t, err)
			tenantProm.Start = overwatch.Start

			value, err := tenantProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
		}
	})

	It("should handle load test", Label("gke", "id=fc171682-00dc-48ee-9686-5eea85890078"), func(ctx context.Context) {
		By(fmt.Sprintf("Installing distributed-chart in namespace %s", namespace))
		install.InstallVMDistributedWithHelm(ctx, vmHelmChart, vmValuesFile, t, namespace, releaseName)

		globalWriteURL := fmt.Sprintf("http://%s/api/v1/write", consts.VMInsertHost(namespace))
		globalReadURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(namespace))

		By("Install Prometheus Benchmark")
		prombenchChartValues := map[string]string{
			"vmtag":                      "v1.133.0",
			"disableMonitoring":          "true",
			"targetsCount":               "500",
			"remoteStorages.vm.writeURL": globalWriteURL,
			"remoteStorages.vm.readURL":  globalReadURL,
		}
		install.InstallPrometheusBenchmark(ctx, t, benchmarkNamespace, prombenchChartValues)

		By("Run 50vus-30mins scenario")
		scenario := "vmselect-50vus-30mins"
		err := install.RunK6Scenario(ctx, t, k6TestsNamespace, vmNamespace, scenario, globalReadURL, 3)
		require.NoError(t, err)

		By("Waiting for K6 jobs to complete")
		install.WaitForK6JobsToComplete(ctx, t, k6TestsNamespace, scenario, 3)

		By("No alerts are firing")
		// overwatch.CheckNoAlertsFiring(ctx, t, vmNamespace, nil)

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
		require.GreaterOrEqual(t, value, float64(4_000))
	})
})
