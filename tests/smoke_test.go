package end_to_end_tests_test

import (
	"fmt"

	"github.com/gruntwork-io/terratest/modules/k8s"

	. "github.com/onsi/ginkgo/v2" //nolint
)

var _ = Describe("Smoke test", Label("vl", "agent", "vlagent"), func() {

	Context("k8s-stack", func() {
		const (
			helmChartName = "vm/victoria-metrics-k8s-stack"
			namespace     = "vm"
			releaseName   = "vmks"
		)

		t := GetT()
		kubectlOptions := k8s.NewKubectlOptions("", "", namespace)

		// k8sClient, err := k8s.GetKubernetesClientFromOptionsE(t, kubectlOptions)
		// require.NoError(t, err)

		// Wait for vmcluster object to change status to operational
		// Wait for vmsingle object to change status to operational
		// Move this to SynchronizedBeforeAll
		// Move defers to SynchronizedAfterAll
		operatorName := fmt.Sprintf("%s-victoria-metrics-operator", releaseName)
		k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, operatorName, retries, pollingInterval)
	})
})
