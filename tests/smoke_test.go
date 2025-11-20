package end_to_end_tests_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/random"

	. "github.com/onsi/ginkgo/v2" //nolint
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

var _ = Describe("Smoke test", Label("vl", "agent", "vlagent"), func() {

	Context("k8s-stack", func() {
		const (
			helmChartName   = "vm/victoria-metrics-k8s-stack"
			smokeValuesFile = "../manifests/smoke.yaml"
		)

		t := GetT()

		namespaceName := fmt.Sprintf("vmstack-%s", strings.ToLower(random.UniqueId()))
		kubectlOptions := k8s.NewKubectlOptions("", "", namespaceName)
		k8s.CreateNamespace(t, kubectlOptions, namespaceName)
		defer k8s.DeleteNamespace(t, kubectlOptions, namespaceName)

		options := &helm.Options{
			BuildDependencies: true,
			KubectlOptions:    kubectlOptions,
			ValuesFiles: []string{
				smokeValuesFile,
			},
		}

		releaseName := fmt.Sprintf("vm-%s", strings.ToLower(random.UniqueId()))
		defer helm.Delete(t, options, releaseName, true)
		helm.Install(t, options, helmChartName, releaseName)

		k8sClient, err := k8s.GetKubernetesClientFromOptionsE(t, kubectlOptions)
		require.NoError(t, err)

		// Verify victoria-metrics-operator components
		operatorName := fmt.Sprintf("%s-victoria-metrics-operator", releaseName)
		k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, operatorName, retries, pollingInterval)

		// Verify kube-state-metrics components
		kubeStateMetricsName := fmt.Sprintf("%s-kube-state-metrics", releaseName)
		k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, kubeStateMetricsName, retries, pollingInterval)
		k8s.WaitUntilServiceAvailable(t, kubectlOptions, kubeStateMetricsName, retries, pollingInterval)

		// Verify prometheus-node-exporter components (DaemonSet)
		promNodeExporterName := fmt.Sprintf("%s-prometheus-node-exporter", releaseName)
		var daemonset *appsv1.DaemonSet
		err = wait.PollUntilContextTimeout(context.Background(), pollingInterval, pollingTimeout, true, func(ctx context.Context) (done bool, err error) {
			daemonset, err = k8sClient.AppsV1().DaemonSets(namespaceName).Get(ctx, promNodeExporterName, metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			return daemonset.Status.CurrentNumberScheduled == daemonset.Status.DesiredNumberScheduled &&
				daemonset.Status.NumberReady == daemonset.Status.DesiredNumberScheduled, nil
		})
		require.NoError(t, err)
		require.NotNil(t, daemonset)
		k8s.WaitUntilServiceAvailable(t, kubectlOptions, promNodeExporterName, retries, pollingInterval)
	})
})
