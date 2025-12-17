package chaos_test

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

func TestChaosTestsTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Chaos test Suite", suiteConfig, reporterConfig)
}

var _ = SynchronizedBeforeSuite(
	func() {
		const (
			chaosValuesFile  = "../../manifests/chaos-mesh-operator/values.yaml"
			chaosReleaseName = "chaos-mesh"
			chaosNamespace   = "chaos-mesh"
			chaosHelmChart   = "chaos-mesh/chaos-mesh"
		)

		ctx := context.Background()
		t := tests.GetT()
		install.DiscoverIngressHost(ctx, t)
		install.InstallChaosMesh(ctx, chaosHelmChart, chaosValuesFile, t, chaosNamespace, chaosReleaseName)

		install.InstallVMGather(t)
		namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
		install.InstallWithHelm(context.Background(), vmHelmChart, vmValuesFile, t, namespace, vmReleaseName)
	}, func() {},
)

const (
	vmReleaseName = "vmks"
	vmHelmChart   = "vm/victoria-metrics-k8s-stack"
	vmValuesFile  = "../../manifests/smoke.yaml"
)

var _ = Describe("Chaos tests", Label("chaos-test"), func() {
	ctx := context.Background()
	t := tests.GetT()

	var overwatch promquery.PrometheusClient

	BeforeEach(func() {
		var err error
		namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())

		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl(namespace))
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl(namespace)))
		require.NoError(t, err)
		overwatch.Start = time.Now()

		// First project should setup victoria-metrics-k8s-stack chart, others will create VMCluster objects
		if GinkgoParallelProcess() != 1 {
			kubeOpts := k8s.NewKubectlOptions("", "", namespace)
			vmclient := install.GetVMClient(t, kubeOpts)

			// Create VMCluster object for other projects
			install.InstallVMCluster(ctx, t, kubeOpts, namespace, vmclient)

			// Ensure VMAgent remote write URL is set up. vmagent always created in vm-1 namespace
			remoteWriteURL := fmt.Sprintf("http://vminsert-%s.%s.svc.cluster.local.:8480/insert/0/prometheus/api/v1/write", namespace, namespace)
			logger.Default.Logf(t, "Setting vmagent remote write URL to %s", remoteWriteURL)
			install.EnsureVMAgentRemoteWriteURL(ctx, t, vmclient, kubeOpts, "vm-1", vmReleaseName, remoteWriteURL)
		}
	})

	AfterEach(func() {
		namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
		defer func() {
			if namespace != "vm-1" {
				kubeOpts := k8s.NewKubectlOptions("", "", namespace)

				install.DeleteVMCluster(t, kubeOpts, namespace)
				k8s.RunKubectl(t, kubeOpts, "delete", "namespace", namespace, "--ignore-not-found=true")
			}
		}()

		gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, namespace)
	})

	Describe("pod restarts", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-pod-failure"), func() {
		scenarios := map[string]string{
			"17f2e31b-9249-4283-845b-aae0bc81e5f2": "vminsert-pod-failure",
			"e340d25f-b14f-4f21-acb4-68c4fdf39a85": "vmstorage-pod-failure",
			"38df1d4b-d38c-4064-8538-c0e03920255f": "vmselect-pod-failure",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func() {
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				By("Run scenario")
				install.RunChaosScenario(ctx, t, namespace, "pods", scenario, "podchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{})
			})
		}
	})

	Describe("cpu stress", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-cpu-stress"), func() {
		scenarios := map[string]string{
			"4c571bca-2442-4a1b-8e54-4f9878f8dd6d": "vminsert-cpu-usage",
			"d1ebdfd3-a0cf-4525-89b9-e998ec7b0c1e": "vmstorage-cpu-usage",
			"f6637d83-be2a-44ab-b446-9c755bad4292": "vmselect-cpu-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func() {
				By("Run scenario")
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				install.RunChaosScenario(ctx, t, namespace, "cpu", scenario, "stresschaos")

				By("Only CPUThrottlingHigh is firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{"CPUThrottlingHigh"})
				overwatch.CheckAlertIsFiring(ctx, t, "CPUThrottlingHigh")
			})
		}
	})

	Describe("memory stress", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-memory-stress"), func() {
		scenarios := map[string]string{
			"47690837-45e5-4cae-9e60-abadf59e4e66": "vminsert-memory-usage",
			"357cef7e-c2ce-4a76-8768-7b142a4e7997": "vmstorage-memory-usage",
			"f9c922b8-104a-4baf-bad3-b00188ccddb1": "vmselect-memory-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func() {
				By("Run scenario")
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				install.RunChaosScenario(ctx, t, namespace, "memory", scenario, "stresschaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{})
			})
		}
	})

	Describe("io stress", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-io-stress"), func() {
		scenarios := map[string]string{
			"c70ce6cc-84fe-447d-8b5f-48871a2ebf99": "vminsert-io-usage",
			"357cef7e-c2ce-4a76-8768-7b142a4e7997": "vmstorage-io-usage",
			"f9c922b8-104a-4baf-bad3-b00188ccddb1": "vmselect-io-usage",
		}

		for uuid, scenario := range scenarios {
			It(fmt.Sprintf("Run %s scenario", scenario), Label(fmt.Sprintf("id=%s", uuid)), func() {
				By("Run scenario")
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				install.RunChaosScenario(ctx, t, namespace, "io", scenario, "stresschaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{})
			})
		}
	})

	Describe("network failure", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-network-failure"), func() {
		networkScenarios := map[string]string{
			"ef3455cd-7687-49a4-b423-7c4541aa051c": "vminsert-to-vmstorage-packet-corrupt",
			"e13108bd-00df-40f5-acc9-b134bc619dc8": "vmselect-to-vmstorage-packet-delay",
			"490c384c-a995-4b46-a5c2-c37baa72beaf": "vmstorage-from-vminsert-packet-loss",
			"260857d8-c49e-4ac3-92e4-220addcc4a53": "vmstorage-from-vmselect-packet-delay",
		}

		for uuid, scenarioName := range networkScenarios {
			It(fmt.Sprintf("Run %s scenario", scenarioName), Label("gke", fmt.Sprintf("id=%s", uuid)), func() {
				By("Run scenario")
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				install.RunChaosScenario(ctx, t, namespace, "network", scenarioName, "networkchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{})
			})
		}

		httpScenarios := map[string]string{
			"98f0368b-b200-4558-a09f-37e7ceaa3b1d": "vminsert-request-delay",
			"d738fdd5-0076-4ddf-9358-2812a9cc3e2b": "vminsert-response-abort",
			"3e1eff4c-dcda-442b-a477-85359ffc57b7": "vmselect-request-delay",
			"b2807243-8528-4500-b630-822ed9fce73d": "vmselect-response-abort",
		}

		for uuid, scenarioName := range httpScenarios {
			It(fmt.Sprintf("Run %s scenario", scenarioName), Label("gke", fmt.Sprintf("id=%s", uuid)), func() {
				By("Run scenario")
				namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
				install.RunChaosScenario(ctx, t, namespace, "http", scenarioName, "httpchaos")

				By("No alerts are firing")
				overwatch.CheckNoAlertsFiring(ctx, t, []string{})
			})
		}
	})

	Describe("rerouting", Ordered, ContinueOnFailure, Label("kind", "gke", "chaos-rerouting"), func() {
		It("Emulate row rerouting when vmstorage-0 becomes unreachable", Label("gke", "id=3a9e309f-eec7-4d37-a7ee-918abd3a3d44"), func() {
			By("Run scenario")
			namespace := fmt.Sprintf("vm-%d", GinkgoParallelProcess())
			scenarioName := "vminsert-to-vmstorage0-3s-delay"
			install.RunChaosScenario(ctx, t, namespace, "network", scenarioName, "NetworkChaos")

			By("No alerts are firing")
			overwatch.CheckNoAlertsFiring(ctx, t, []string{})
		})
	})
})
