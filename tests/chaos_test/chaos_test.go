package chaos_test

import (
	"context"
	"fmt"
	"os/exec"
	"testing"
	"time"

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
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "Chaos test Suite", suiteConfig, reporterConfig)
}

var _ = Describe("Chaos tests", Ordered, Label("chaos-test"), func() {
	const (
		vmNamespace   = "vm"
		vmReleaseName = "vmks"
		vmHelmChart   = "vm/victoria-metrics-k8s-stack"
		vmValuesFile  = "../../manifests/smoke.yaml"

		chaosValuesFile  = "../../manifests/chaos-mesh-operator/values.yaml"
		chaosReleaseName = "chaos-mesh"
		chaosNamespace   = "chaos-mesh"
		chaosHelmChart   = "chaos-mesh/chaos-mesh"
	)

	ctx := context.Background()
	ctxCancel, cancel := context.WithCancel(ctx)
	AfterAll(func() {
		cancel()
	})
	t := tests.GetT()

	overwatch, err := promquery.NewPrometheusClient("http://localhost:8429/prometheus")
	require.NoError(t, err)

	BeforeAll(func() {
		overwatch.Start = time.Now()
		install.InstallWithHelm(ctx, vmHelmChart, vmValuesFile, t, vmNamespace, vmReleaseName)

		// Install chaos-mesh operator
		install.InstallChaosMesh(ctx, chaosHelmChart, chaosValuesFile, t, chaosNamespace, chaosReleaseName)
	})
	AfterEach(func() {
		gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)
		gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout, vmNamespace)
	})

	It("Run vminsert-pod-failure scenario", Label("kind", "gke", "id=17f2e31b-9249-4283-845b-aae0bc81e5f2"), func() {
		By("Run scenario")
		namespace := "vm"
		install.RunChaosScenario(ctx, t, namespace, "pods", "vminsert-pod-failure", "podchaos")

		By("Setup port-forwarding for overwatch")
		cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmsingle-overwatch", "8429:8429")
		go func() {
			stdoutStderr, err := cmd.CombinedOutput()
			logger.Default.Logf(t, "overwatch port-forward output: %s", stdoutStderr)
			logger.Default.Logf(t, "overwatch port-forward err: %v", err)
		}()
		// Hack: give it some time to start
		time.Sleep(10 * time.Second)

		By("No alerts are firing")
		value, err := overwatch.VectorValue(ctx, `sum by (alertname) (vmalert_alerts_firing{alertname!~"(InfoInhibitor|Watchdog|TooManyLogs|RecordingRulesError|AlertingRulesError)"})`)
		// require.NoError(t, err)
		require.Zero(t, value)

		By("No services were down")
		value, err = overwatch.VectorValue(ctx, "min_over_time(up) == 0")
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, float64(1))
	})

	networkScenarios := map[string]string{
		"148c9b15-7779-414e-9f99-9a92e54b6816": "vmagent-to-vminsert-packet-delay",
		"f767bbe7-b84c-4c37-8bf3-eaf3f6e34909": "vmagent-to-vminsert-packet-loss",
		"238a19bd-e674-4359-a4ee-c00421014f67": "vminsert-from-vmagent-packet-delay",
		"0d9545bb-c1c6-4a03-856c-8ec187d581a9": "vminsert-from-vmagent-packet-loss",
		"ef3455cd-7687-49a4-b423-7c4541aa051c": "vminsert-to-vmstorage-packet-corrupt",
		"070679cc-3ba7-41a2-9c41-94bd9d1f61ba": "vminsert-to-vmstorage-packet-delay",
		"cfd198c1-f307-4366-9301-530384d68190": "vminsert-to-vmstorage-packet-loss",
		"b8364e50-4c2e-412d-8896-3c350cdef31a": "vmselect-to-vmstorage-packet-corrupt",
		"e13108bd-00df-40f5-acc9-b134bc619dc8": "vmselect-to-vmstorage-packet-delay",
		"8343989e-34a9-4469-bd4c-c6900e3c5a11": "vmselect-to-vmstorage-packet-loss",
		"a8a00f36-18b0-42c0-ae0a-f14a7a5a08c7": "vmstorage-from-vminsert-packet-corrupt",
		"1321ea2f-a0fa-4fd9-8bd4-57f6d3a636c4": "vmstorage-from-vminsert-packet-delay",
		"490c384c-a995-4b46-a5c2-c37baa72beaf": "vmstorage-from-vminsert-packet-loss",
		"12f1fa4e-f454-4942-b73c-3df1116daea2": "vmstorage-from-vmselect-packet-corrupt",
		"260857d8-c49e-4ac3-92e4-220addcc4a53": "vmstorage-from-vmselect-packet-delay",
		"63b77044-a445-49fc-9deb-96c32ccbcde2": "vmstorage-from-vmselect-packet-loss",
	}

	for uuid, scenarioName := range networkScenarios {
		It(fmt.Sprintf("Run %s scenario", scenarioName), Label("gke", fmt.Sprintf("id=%s", uuid)), func() {
			By("Run scenario")
			namespace := "vm"
			install.RunChaosScenario(ctx, t, namespace, "network", scenarioName, "NetworkChaos")

			By("No alerts are firing")
			value, _ := overwatch.VectorValue(ctx, `sum by (alertname) (vmalert_alerts_firing{alertname!~"(InfoInhibitor|Watchdog|TooManyLogs|RecordingRulesError|AlertingRulesError)"})`)
			// require.NoError(t, err)
			require.Zero(t, value)

			By("No services were down")
			value, err = overwatch.VectorValue(ctx, "min_over_time(up) == 0")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(1))
		})
	}

	// Requires proper CNI in the cluster
	It("Run vminsert-request-abort scenario", Label("gke", "id=2195bf4c-7dca-4bb1-a363-89dbc898a507"), func() {
		By("Run scenario")
		namespace := "vm"
		scenarioName := "vminsert-request-abort"
		install.RunChaosScenario(ctx, t, namespace, "http", scenarioName, "NetworkChaos")

		By("No alerts are firing")
		value, err := overwatch.VectorValue(ctx, `sum by (alertname) (vmalert_alerts_firing{alertname!~"(InfoInhibitor|Watchdog|TooManyLogs|RecordingRulesError|AlertingRulesError)"})`)
		require.NoError(t, err)
		require.Zero(t, value)

		By("No alerts are firing")
		value, err = overwatch.VectorValue(ctx, "min_over_time(up) == 0")
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, float64(1))
	})
})
