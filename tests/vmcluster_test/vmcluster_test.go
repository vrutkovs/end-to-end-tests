package vmcluster_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/VictoriaMetrics/lib/prompb"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/remotewrite"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestVMClusterTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "VMCluster test Suite", suiteConfig, reporterConfig)
}

var (
	t         terratesting.TestingT
	namespace string
	overwatch promquery.PrometheusClient
	c         *http.Client
)

const (
	releaseName        = "vmks"
	overwatchNamespace = "overwatch"
	vmNamespace        = "monitoring"
	vmHelmChart        = "vm/victoria-metrics-k8s-stack"
	vmValuesFile       = "../../manifests/smoke.yaml"
)

// Install VM from helm chart for the first process, set namespace for the rest
var _ = SynchronizedBeforeSuite(
	func(ctx context.Context) {
		t = tests.GetT()
		install.DiscoverIngressHost(ctx, t)
		install.InstallVMGather(t)
		install.InstallWithHelm(context.Background(), vmHelmChart, vmValuesFile, t, vmNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, vmNamespace, releaseName)

		// Remove stock VMCluster - it would be recreated in vm* namespaces
		kubeOpts := k8s.NewKubectlOptions("", "", vmNamespace)
		install.DeleteVMCluster(t, kubeOpts, releaseName)
	}, func(ctx context.Context) {
		t = tests.GetT()
		namespace = fmt.Sprintf("vm%d", GinkgoParallelProcess())
	},
)

var _ = Describe("VMCluster test", Label("vmcluster"), func() {
	BeforeEach(func(ctx context.Context) {
		install.DiscoverIngressHost(ctx, t)
		var err error

		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl())
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl()))
		require.NoError(t, err)
		overwatch.Start = time.Now()

		// Create new VMCluster object
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)
		vmclient := install.GetVMClient(t, kubeOpts)
		install.InstallVMCluster(ctx, t, kubeOpts, namespace, vmclient)
		c = &http.Client{
			Timeout: time.Second * 10,
		}
	})

	AfterEach(func(ctx context.Context) {
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)

		install.DeleteVMCluster(t, kubeOpts, namespace)
		k8s.RunKubectl(t, kubeOpts, "delete", "namespace", namespace, "--ignore-not-found=true")

		if CurrentSpecReport().Failed() {
			gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, releaseName)
			gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		}
	})

	Describe("Multitenancy", func() {
		It("should not mix data sent to different tenants", Label("gke", "id=66618081-b150-4b48-8180-ae1f53512117"), func(ctx context.Context) {
			By("Inserting data into tenant 0")
			tenantOneInsertURL := fmt.Sprintf("http://%s/insert/0/prometheus/api/v1/write", consts.VMInsertHost(namespace))
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			err := remotewrite.RemoteWrite(c, ts, tenantOneInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			tenantTwoInsertURL := fmt.Sprintf("http://%s/insert/1/prometheus/api/v1/write", consts.VMInsertHost(namespace))
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			err = remotewrite.RemoteWrite(c, ts, tenantTwoInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data is not mixed")
			tenantOneSelectURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(namespace))
			tenantOneProm, err := promquery.NewPrometheusClient(tenantOneSelectURL)
			require.NoError(t, err)
			tenantOneProm.Start = overwatch.Start

			value, err := tenantOneProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = tenantOneProm.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			tenantTwoSelectURL := fmt.Sprintf("%s/select/1/prometheus", consts.VMSelectUrl(namespace))
			tenantTwoProm, err := promquery.NewPrometheusClient(tenantTwoSelectURL)
			require.NoError(t, err)
			tenantTwoProm.Start = overwatch.Start

			value, err = tenantTwoProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
			value, err = tenantTwoProm.VectorValue(ctx, "foo_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			By("Verifying data can be retrieved via multitenant URL")
			multitenantSelectURL := fmt.Sprintf("%s/select/multitenant/prometheus", consts.VMSelectUrl(namespace))
			multitenantProm, err := promquery.NewPrometheusClient(multitenantSelectURL)
			require.NoError(t, err)
			multitenantProm.Start = overwatch.Start

			value, err = multitenantProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = multitenantProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
		})

		It("should accept data via multitenant URL", Label("gke", "id=16c08934-9e25-45ed-a94b-4fbbbe3170ef"), func(ctx context.Context) {
			multitenantInsertURL := fmt.Sprintf("http://%s/insert/multitenant/prometheus/api/v1/write", consts.VMInsertHost(namespace))

			By("Inserting data into tenant 0")
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			for i, item := range ts {
				ts[i].Labels = append(item.Labels, prompb.Label{
					Name: "vm_account_id", Value: "0",
				})
			}
			err := remotewrite.RemoteWrite(c, ts, multitenantInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			for i, item := range ts {
				ts[i].Labels = append(item.Labels, prompb.Label{
					Name: "vm_account_id", Value: "1",
				})
			}
			err = remotewrite.RemoteWrite(c, ts, multitenantInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data is not mixed")
			tenantOneSelectURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(namespace))
			tenantOneProm, err := promquery.NewPrometheusClient(tenantOneSelectURL)
			require.NoError(t, err)
			tenantOneProm.Start = overwatch.Start

			value, err := tenantOneProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = tenantOneProm.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			tenantTwoSelectURL := fmt.Sprintf("%s/select/1/prometheus", consts.VMSelectUrl(namespace))
			tenantTwoProm, err := promquery.NewPrometheusClient(tenantTwoSelectURL)
			require.NoError(t, err)
			tenantTwoProm.Start = overwatch.Start

			value, err = tenantTwoProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
			value, err = tenantTwoProm.VectorValue(ctx, "foo_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			By("Verifying data can be retrieved via multitenant URL")
			multitenantSelectURL := fmt.Sprintf("%s/select/multitenant/prometheus", consts.VMSelectUrl(vmNamespace))
			multitenantProm, err := promquery.NewPrometheusClient(multitenantSelectURL)
			multitenantProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err = multitenantProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = multitenantProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
		})

		It("should accept data via multitenant URL", Label("gke", "id=16c08934-9e25-45ed-a94b-4fbbbe3170ef"), func(ctx context.Context) {
			multitenantInsertURL := fmt.Sprintf("http://%s/insert/multitenant/prometheus/api/v1/write", consts.VMInsertHost(vmNamespace))

			By("Inserting data into tenant 0")
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			for i, item := range ts {
				ts[i].Labels = append(item.Labels, prompb.Label{
					Name: "vm_account_id", Value: "0",
				})
			}
			err := remotewrite.RemoteWrite(c, ts, multitenantInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			for i, item := range ts {
				ts[i].Labels = append(item.Labels, prompb.Label{
					Name: "vm_account_id", Value: "1",
				})
			}
			err = remotewrite.RemoteWrite(c, ts, multitenantInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data is not mixed")
			tenantOneSelectURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(vmNamespace))
			tenantOneProm, err := promquery.NewPrometheusClient(tenantOneSelectURL)
			tenantOneProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err := tenantOneProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = tenantOneProm.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			tenantTwoSelectURL := fmt.Sprintf("%s/select/1/prometheus", consts.VMSelectUrl(vmNamespace))
			tenantTwoProm, err := promquery.NewPrometheusClient(tenantTwoSelectURL)
			tenantTwoProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err = tenantTwoProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
			value, err = tenantTwoProm.VectorValue(ctx, "foo_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))
		})

		It("should retrieve data from different tenants via multitenant URL", Label("gke", "id=7e075898-f6c4-49d5-9d7f-8a6163759065"), func(ctx context.Context) {
			By("Inserting data into tenant 0")
			tenantOneInsertURL := fmt.Sprintf("http://%s/insert/0/prometheus/api/v1/write", consts.VMInsertHost(vmNamespace))
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			err := remotewrite.RemoteWrite(c, ts, tenantOneInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			tenantTwoInsertURL := fmt.Sprintf("http://%s/insert/1/prometheus/api/v1/write", consts.VMInsertHost(vmNamespace))
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			err = remotewrite.RemoteWrite(c, ts, tenantTwoInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data can be retrieved via multitenant URL")
			multitenantSelectURL := fmt.Sprintf("%s/select/multitenant/prometheus", consts.VMSelectUrl(vmNamespace))
			multitenantProm, err := promquery.NewPrometheusClient(multitenantSelectURL)
			multitenantProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err := multitenantProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = multitenantProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
		})

		It("should retrieve data from different tenants via multitenant URL", Label("gke", "id=7e075898-f6c4-49d5-9d7f-8a6163759065"), func(ctx context.Context) {
			By("Inserting data into tenant 0")
			tenantOneInsertURL := fmt.Sprintf("http://%s/insert/0/prometheus/api/v1/write", consts.VMInsertHost(namespace))
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			err := remotewrite.RemoteWrite(c, ts, tenantOneInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			tenantTwoInsertURL := fmt.Sprintf("http://%s/insert/1/prometheus/api/v1/write", consts.VMInsertHost(namespace))
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			err = remotewrite.RemoteWrite(c, ts, tenantTwoInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data can be retrieved via multitenant URL")
			multitenantSelectURL := fmt.Sprintf("%s/select/multitenant/prometheus", consts.VMSelectUrl(namespace))
			multitenantProm, err := promquery.NewPrometheusClient(multitenantSelectURL)
			require.NoError(t, err)
			multitenantProm.Start = overwatch.Start

			value, err := multitenantProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = multitenantProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
		})
	})
})
