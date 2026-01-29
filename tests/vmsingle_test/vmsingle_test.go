package vmsingle_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	jsonpatch "github.com/evanphx/json-patch/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

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

func TestVMSingleTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "VMSingle test Suite", suiteConfig, reporterConfig)
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
		install.InstallVMK8StackWithHelm(context.Background(), vmHelmChart, vmValuesFile, t, vmNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, vmNamespace, releaseName)

		// Remove stock VMCluster
		kubeOpts := k8s.NewKubectlOptions("", "", vmNamespace)
		install.DeleteVMCluster(t, kubeOpts, releaseName)

	}, func(ctx context.Context) {
		t = tests.GetT()
		namespace = fmt.Sprintf("vm%d", GinkgoParallelProcess())
	},
)

var _ = Describe("VMSingle test", Label("vmsingle"), func() {
	BeforeEach(func(ctx context.Context) {
		install.DiscoverIngressHost(ctx, t)
		var err error

		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl())
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl()))
		require.NoError(t, err)
		overwatch.Start = time.Now()

		c = &http.Client{
			Timeout: time.Second * 10,
		}
	})

	AfterEach(func(ctx context.Context) {
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)

		install.DeleteVMSingle(t, kubeOpts, namespace)
		k8s.RunKubectl(t, kubeOpts, "delete", "namespace", namespace, "--ignore-not-found=true")

		if CurrentSpecReport().Failed() {
			gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, releaseName)
			gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		}
	})

	Describe("Relabeling", func() {
		It("should relabel data sent via remote write", Label("gke", "id=e72f26ba-c1b7-4671-9c7e-7cfa630c33a9"), func(ctx context.Context) {
			kubeOpts := k8s.NewKubectlOptions("", "", namespace)
			if _, err := k8s.GetNamespaceE(t, kubeOpts, namespace); err != nil {
				k8s.CreateNamespace(t, kubeOpts, namespace)
			}

			By("Configure VMSingle to relabel data")
			// Create configmap
			cfgMapName := "vmsingle-relabel-config"
			configMap := &corev1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "ConfigMap",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: cfgMapName,
				},
				Data: map[string]string{
					"relabel.yml": `
- target_label: cluster
  replacement: dev
- action: drop
  source_labels: [__name__]
  regex: bar_.*
`,
				},
			}
			resource, err := runtime.DefaultUnstructuredConverter.ToUnstructured(configMap)
			require.NoError(t, err)
			cfgMapBytes, err := yaml.Marshal(resource)
			require.NoError(t, err)
			k8s.KubectlApplyFromString(t, kubeOpts, string(cfgMapBytes))

			patchContent := []byte(fmt.Sprintf(`[
				{"op": "add", "path": "/spec/extraArgs", "value": {}},
				{"op": "add", "path": "/spec/extraArgs/relabelConfig", "value": "/etc/vm/configs/%s/relabel.yml"},
				{"op": "add", "path": "/spec/configMaps", "value": []},
				{"op": "add", "path": "/spec/configMaps/-", "value": "%s"}
			]`, cfgMapName, cfgMapName))
			patch, err := jsonpatch.DecodePatch(patchContent)
			require.NoError(t, err)

			vmclient := install.GetVMClient(t, kubeOpts)
			install.InstallVMSingle(ctx, t, kubeOpts, namespace, vmclient, []jsonpatch.Patch{patch})

			By("Inserting data into tenant 0")
			tenantOneInsertURL := fmt.Sprintf("http://%s/api/v1/write", consts.VMSingleNamespacedHost(namespace))
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			err = remotewrite.RemoteWrite(c, ts, tenantOneInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			tenantTwoInsertURL := fmt.Sprintf("http://%s/api/v1/write", consts.VMSingleNamespacedHost(namespace))
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			err = remotewrite.RemoteWrite(c, ts, tenantTwoInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("foo has cluster=dev label")
			tenantOneSelectURL := fmt.Sprintf("http://%s/prometheus", consts.VMSingleNamespacedHost(namespace))
			tenantOneProm, err := promquery.NewPrometheusClient(tenantOneSelectURL)
			require.NoError(t, err)
			tenantOneProm.Start = overwatch.Start

			value, err := tenantOneProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))

			labels, err := tenantOneProm.VectorMetric(ctx, "foo_2")
			require.NoError(t, err)
			require.Contains(t, labels, model.LabelName("cluster"))
			require.Equal(t, labels["cluster"], model.LabelValue("dev"))

			By("bar_2 was removed")
			value, err = tenantOneProm.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))
		})
	})
})
