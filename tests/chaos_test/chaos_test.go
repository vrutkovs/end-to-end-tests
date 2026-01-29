package chaos_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"

	jsonpatch "github.com/evanphx/json-patch/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestChaosTestsTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Chaos test Suite", suiteConfig, reporterConfig)
}

const (
	chaosValuesFile  = "../../manifests/chaos-mesh-operator/values.yaml"
	chaosReleaseName = "chaos-mesh"
	chaosNamespace   = "chaos-mesh"
	chaosHelmChart   = "chaos-mesh/chaos-mesh"

	releaseName        = "vmks"
	overwatchNamespace = "overwatch"
	k8sStackNamespace  = "monitoring"
	vmHelmChart        = "vm/victoria-metrics-k8s-stack"
	vmValuesFile       = "../../manifests/smoke.yaml"
)

var (
	t         terratesting.TestingT
	namespace string
	overwatch promquery.PrometheusClient
)

// Install VM from helm chart for the first process, set namespace for the rest
var _ = SynchronizedBeforeSuite(
	func(ctx context.Context) {
		t := tests.GetT()
		install.DiscoverIngressHost(ctx, t)
		install.InstallChaosMesh(ctx, chaosHelmChart, chaosValuesFile, t, chaosNamespace, chaosReleaseName)
		install.InstallVMGather(t)
		install.InstallVMK8StackWithHelm(context.Background(), vmHelmChart, vmValuesFile, t, k8sStackNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, k8sStackNamespace, releaseName)

		// Remove stock VMCluster - it would be recreated in vm* namespaces
		kubeOpts := k8s.NewKubectlOptions("", "", k8sStackNamespace)
		install.DeleteVMCluster(t, kubeOpts, releaseName)
	}, func(ctx context.Context) {
		t = tests.GetT()
		namespace = fmt.Sprintf("vm%d", GinkgoParallelProcess())
	},
)

var _ = Describe("Chaos tests", Label("chaos-test"), func() {
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
		install.InstallVMCluster(ctx, t, kubeOpts, namespace, vmclient, []jsonpatch.Patch{})

		// Ensure VMAgent remote write URL is set up. vmagent already created in k8sStackNamespace namespace
		remoteWriteURL := fmt.Sprintf("http://vminsert-%s.%s.svc.cluster.local.:8480/insert/0/prometheus/api/v1/write", namespace, namespace)
		logger.Default.Logf(t, "Setting vmagent remote write URL to %s", remoteWriteURL)
		install.EnsureVMAgentRemoteWriteURL(ctx, t, vmclient, kubeOpts, k8sStackNamespace, releaseName, remoteWriteURL)
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

	Describe("pod restarts", Label("kind", "gke", "chaos-pod-failure"), func() {
		scenarios := map[string]string{
			"17f2e31b-9249-4283-845b-aae0bc81e5f2": "vminsert-pod-failure",
			"e340d25f-b14f-4f21-acb4-68c4fdf39a85": "vmstorage-pod-failure",
			"38df1d4b-d38c-4064-8538-c0e03920255f": "vmselect-pod-failure",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "pods", scenario, "podchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}
	})

	Describe("cpu stress", Label("kind", "gke", "chaos-cpu-stress"), func() {
		scenarios := map[string]string{
			"4c571bca-2442-4a1b-8e54-4f9878f8dd6d": "vminsert-cpu-usage",
			"d1ebdfd3-a0cf-4525-89b9-e998ec7b0c1e": "vmstorage-cpu-usage",
			"f6637d83-be2a-44ab-b446-9c755bad4292": "vmselect-cpu-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "cpu", scenario, "stresschaos")

				By("Only CPUThrottlingHigh is firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, []string{"CPUThrottlingHigh"})
				overwatch.CheckAlertIsFiring(ctx, t, namespace, "CPUThrottlingHigh")

				// By("No alerts are firing")
				// overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}
	})

	Describe("memory stress", Label("kind", "gke", "chaos-memory-stress"), func() {
		scenarios := map[string]string{
			"47690837-45e5-4cae-9e60-abadf59e4e66": "vminsert-memory-usage",
			"357cef7e-c2ce-4a76-8768-7b142a4e7997": "vmstorage-memory-usage",
			"f9c922b8-104a-4baf-bad3-b00188ccddb1": "vmselect-memory-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "memory", scenario, "stresschaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}
	})

	Describe("io stress", Label("kind", "gke", "chaos-io-stress"), func() {
		scenarios := map[string]string{
			"c70ce6cc-84fe-447d-8b5f-48871a2ebf99": "vminsert-io-usage",
			"357cef7e-c2ce-4a76-8768-7b142a4e7997": "vmstorage-io-usage",
			"f9c922b8-104a-4baf-bad3-b00188ccddb1": "vmselect-io-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "io", scenario, "stresschaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}
	})

	Describe("network failure", Label("kind", "gke", "chaos-network-failure"), func() {
		networkScenarios := map[string]string{
			"ef3455cd-7687-49a4-b423-7c4541aa051c": "vminsert-to-vmstorage-packet-corrupt",
			"e13108bd-00df-40f5-acc9-b134bc619dc8": "vmselect-to-vmstorage-packet-delay",
			"490c384c-a995-4b46-a5c2-c37baa72beaf": "vmstorage-from-vminsert-packet-loss",
			"260857d8-c49e-4ac3-92e4-220addcc4a53": "vmstorage-from-vmselect-packet-delay",
		}

		for uuid, scenarioName := range networkScenarios {
			It(fmt.Sprintf("Run %s scenario", scenarioName), Label("gke", fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "network", scenarioName, "networkchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}

		httpScenarios := map[string]string{
			"98f0368b-b200-4558-a09f-37e7ceaa3b1d": "vminsert-request-delay",
			"d738fdd5-0076-4ddf-9358-2812a9cc3e2b": "vminsert-response-abort",
			"3e1eff4c-dcda-442b-a477-85359ffc57b7": "vmselect-request-delay",
			"b2807243-8528-4500-b630-822ed9fce73d": "vmselect-response-abort",
		}

		for uuid, scenarioName := range httpScenarios {
			It(fmt.Sprintf("Run %s scenario", scenarioName), Label("gke", fmt.Sprintf("id=%s", uuid)), func(ctx context.Context) {
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "http", scenarioName, "httpchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
			})
		}
	})

	// Describe("rerouting", Label("kind", "gke", "chaos-rerouting"), func() {
	// 	It("Emulate row rerouting when vmstorage-0 becomes unreachable", Label("gke", "id=3a9e309f-eec7-4d37-a7ee-918abd3a3d44"), func(ctx context.Context) {
	// 		By("Run scenario")
	// 		scenarioName := "vminsert-to-vmstorage0-3s-delay"
	// 		install.RunChaosScenario(ctx, t, namespace, "network", scenarioName, "networkchaos")

	// 		By("No alerts are firing")
	// 		overwatch.CheckNoAlertsFiring(ctx, t, namespace, nil)
	// 	})
	// })
})
