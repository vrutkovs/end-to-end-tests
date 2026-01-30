package vmsingle_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestVMSingleTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	// suiteConfig.FocusStrings = []string{"should aggregate data with sum_samples output"}
	RunSpecs(t, "VMSingle test Suite", suiteConfig, reporterConfig)
}

var (
	t         terratesting.TestingT
	namespace string
	overwatch promquery.PrometheusClient
	c         *http.Client
)

// Install VM from helm chart for the first process, set namespace for the rest
var _ = SynchronizedBeforeSuite(
	func(ctx context.Context) {
		t = tests.GetT()
		install.DiscoverIngressHost(ctx, t)
		install.InstallVMGather(t)
		install.InstallVMK8StackWithHelm(
			context.Background(),
			consts.VMK8sStackChart,
			consts.SmokeValuesFile,
			t,
			consts.DefaultVMNamespace,
			consts.DefaultReleaseName,
		)
		install.InstallOverwatch(ctx, t, consts.OverwatchNamespace, consts.DefaultVMNamespace, consts.DefaultReleaseName)

		// Remove stock VMCluster
		kubeOpts := k8s.NewKubectlOptions("", "", consts.DefaultVMNamespace)
		install.DeleteVMCluster(t, kubeOpts, consts.DefaultReleaseName)
	}, func(ctx context.Context) {
		t = tests.GetT()
		namespace = tests.ParallelNamespace("vm")
	},
)

var _ = Describe("VMSingle test", Label("vmsingle"), func() {
	BeforeEach(func(ctx context.Context) {
		var err error
		overwatch, err = tests.SetupOverwatchClient(ctx, t)
		require.NoError(t, err)

		c = tests.NewHTTPClient()
	})

	AfterEach(func(ctx context.Context) {
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)

		install.DeleteVMSingle(t, kubeOpts, namespace)
		tests.CleanupNamespace(t, kubeOpts, namespace)
		tests.GatherOnFailure(ctx, t, consts.DefaultReleaseName)
	})

	Describe("Relabeling", func() {
		It("should relabel data sent via remote write", Label("gke", "id=aabbccdd-eeff-0011-2233-445566778899"), func(ctx context.Context) {
			kubeOpts := k8s.NewKubectlOptions("", "", namespace)
			tests.EnsureNamespaceExists(t, kubeOpts, namespace)

			By("Configure VMSingle to relabel data")
			cfgMapName := "vmsingle-relabel-config"

			// Build relabel config using builder
			relabelConfig := tests.NewRelabelConfigBuilder().
				AddLabel("cluster", "dev").
				DropByName("bar_.*").
				MustBuild()

			// Build and apply ConfigMap using builder
			err := tests.NewConfigMapBuilder(cfgMapName).
				WithRelabelConfig(relabelConfig).
				Apply(t, kubeOpts)
			require.NoError(t, err)

			// Build JSON patch using builder
			patch := tests.NewJSONPatchBuilder().
				WithVMSingleConfig(cfgMapName, "relabelConfig", "relabel.yml").
				MustBuild()

			vmclient := install.GetVMClient(t, kubeOpts)
			install.InstallVMSingle(ctx, t, kubeOpts, namespace, vmclient, []jsonpatch.Patch{patch})

			// Build remote write helper
			remoteWriter := tests.NewRemoteWriteBuilder().
				WithHTTPClient(c).
				ForVMSingle(namespace)

			By("Inserting foo data (should be relabeled)")
			fooTimeSeries := tests.NewTimeSeriesBuilder("foo").
				WithCount(10).
				WithValue(1).
				Build()
			err = remoteWriter.Send(fooTimeSeries)
			require.NoError(t, err)

			By("Inserting bar data (should be dropped)")
			barTimeSeries := tests.NewTimeSeriesBuilder("bar").
				WithCount(10).
				WithValue(5).
				Build()
			err = remoteWriter.Send(barTimeSeries)
			require.NoError(t, err)

			tests.WaitForDataPropagation()

			By("foo has cluster=dev label")
			prom := tests.NewPromClientBuilder().
				ForVMSingle(namespace).
				WithStartTime(overwatch.Start).
				MustBuild()

			value, err := prom.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))

			labels, err := prom.VectorMetric(ctx, "foo_2")
			require.NoError(t, err)
			require.Contains(t, labels, model.LabelName("cluster"))
			require.Equal(t, labels["cluster"], model.LabelValue("dev"))

			By("bar_2 was removed")
			value, err = prom.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, consts.ErrNoDataReturned)
			require.Equal(t, value, model.SampleValue(0))
		})
	})

	Describe("Streaming Aggregation", func() {
		It("should aggregate data with sum_samples output", Label("gke", "id=a1b2c3d4-e5f6-7890-abcd-ef1234567890"), func(ctx context.Context) {
			kubeOpts := k8s.NewKubectlOptions("", "", namespace)
			tests.EnsureNamespaceExists(t, kubeOpts, namespace)

			By("Configure VMSingle with streaming aggregation")
			cfgMapName := "vmsingle-stream-aggr-config"

			// Build streaming aggregation config using builder
			streamAggrConfig := tests.NewStreamAggrConfigBuilder().
				AddRule(`{__name__=~"aggr_.*"}`, "30s", []string{"sum_samples"}).
				WithoutLabels("foo", "bar", "baz").
				MustBuild()

			// Build and apply ConfigMap using builder
			err := tests.NewConfigMapBuilder(cfgMapName).
				WithStreamAggrConfig(streamAggrConfig).
				Apply(t, kubeOpts)
			require.NoError(t, err)

			// Build JSON patch using builder
			patch := tests.NewJSONPatchBuilder().
				WithVMSingleConfig(cfgMapName, "streamAggr.config", "stream-aggr.yml").
				MustBuild()

			vmclient := install.GetVMClient(t, kubeOpts)
			install.InstallVMSingle(ctx, t, kubeOpts, namespace, vmclient, []jsonpatch.Patch{patch})

			// Build remote write helper
			remoteWriter := tests.NewRemoteWriteBuilder().
				WithHTTPClient(c).
				ForVMSingle(namespace)

			By("Inserting multiple samples for aggregation")
			for i := 0; i < 5; i++ {
				aggrTimeSeries := tests.NewTimeSeriesBuilder("aggr_test").
					WithCount(3).
					WithValue(1).
					Build()
				err = remoteWriter.Send(aggrTimeSeries)
				require.NoError(t, err)
				time.Sleep(2 * time.Second)
			}

			By("Inserting non-matching metrics")
			nonAggrTimeSeries := tests.NewTimeSeriesBuilder("nonaggr").
				WithCount(3).
				WithValue(100).
				Build()
			err = remoteWriter.Send(nonAggrTimeSeries)
			require.NoError(t, err)

			By("Waiting for aggregation interval to pass")
			tests.WaitForAggregation()

			By("Verifying aggregated metrics exist with correct naming")
			prom := tests.NewPromClientBuilder().
				ForVMSingle(namespace).
				WithStartTime(overwatch.Start).
				MustBuild()

			value, err := prom.VectorValue(ctx, "sum_over_time(aggr_test_0:30s_without_bar_baz_foo_sum_samples[5m])")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))

			By("Verifying non-matching metrics are written as-is")
			value, err = prom.VectorValue(ctx, "nonaggr_0")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(100))

			By("Verifying original aggr metrics are dropped")
			value, err = prom.VectorValue(ctx, "aggr_test_0")
			require.EqualError(t, err, consts.ErrNoDataReturned)
			require.Equal(t, value, model.SampleValue(0))
		})
	})
})
