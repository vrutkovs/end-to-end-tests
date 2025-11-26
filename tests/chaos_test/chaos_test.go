package chaos_test

import (
	"context"
	"testing"
	"time"

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
	suiteConfig.LabelFilter = "chaos-test"
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
	t := tests.GetT()

	overwatch, err := promquery.NewPrometheusClient("http://localhost:8481/select/0/prometheus")
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

	It("Run vminsert-pod-failure scenario", Label("id=17f2e31b-9249-4283-845b-aae0bc81e5f2"), func() {
		By("Run scenario")
		namespace := "vm"
		install.RunChaosScenario(ctx, t, namespace, "pods", "vminsert-pod-failure", "PodChaos")

		By("No alerts are firing")
		value, err := overwatch.VectorValue(ctx, "min_over_time(up) == 0")
		require.NoError(t, err)
		require.GreaterOrEqual(t, value, float64(1))
	})

	// Requires proper CNI in the cluster
	// It("Run vminsert-request-abort scenario", Label("id=2195bf4c-7dca-4bb1-a363-89dbc898a507"), func() {
	// 	By("Run scenario")
	// 	scenarioFolder := "http"
	// 	scenario := "vminsert-request-abort"
	// 	err := install.RunChaosScenario(ctx, t, scenarioFolder, scenario, "HTTPChaos")
	// 	require.NoError(t, err)

	// 	By("No alerts are firing")
	// 	value, err := overwatch.VectorValue(ctx, "min_over_time(up) == 0")
	// 	require.NoError(t, err)
	// 	require.GreaterOrEqual(t, value, float64(1))
	// })
})
