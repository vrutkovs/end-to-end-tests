package install

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
)

func InstallChaosMesh(ctx context.Context, helmChart, valuesFile string, t terratesting.TestingT, namespace string, releaseName string) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	helmOpts := &helm.Options{
		KubectlOptions: kubeOpts,
		ValuesFiles:    []string{valuesFile},
		ExtraArgs: map[string][]string{
			"upgrade": {"--create-namespace", "--wait"},
		},
	}

	By(fmt.Sprintf("Install %s chart", helmChart))
	helm.Upgrade(t, helmOpts, helmChart, releaseName)

	// Install ebtables on the node
	manifestPath := "../../manifests/chaos-mesh-operator/ebtables.yaml"
	k8s.KubectlApply(t, kubeOpts, manifestPath)

	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "chaos-controller-manager", consts.Retries, consts.PollingInterval)
}

func RunChaosScenario(ctx context.Context, t terratesting.TestingT, namespace, scenarioFolder, scenario, chaosType string) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)

	kubeConfigPath, err := kubeOpts.GetConfigPath(t)
	require.NoError(t, err)
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	require.NoError(t, err)

	dynamicClient := dynamic.NewForConfigOrDie(restConfig)
	require.NoError(t, err)

	manifestPath := fmt.Sprintf("../../manifests/chaos-tests/%s/%s.yaml", scenarioFolder, scenario)
	k8s.KubectlApply(t, kubeOpts, manifestPath)

	By("Waiting for chaos scenario to complete")
	WaitForChaosScenarioToComplete(ctx, t, dynamicClient, namespace, scenario, chaosType)
}

func WaitForChaosScenarioToComplete(ctx context.Context, t terratesting.TestingT, chaosClient *dynamic.DynamicClient, namespace, scenario, chaosType string) {
	gvr := schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1alpha1", Resource: chaosType}

	chaosTestOverCtx, cancel := context.WithTimeout(ctx, consts.ChaosTestMaxDuration)
	defer cancel()

	ticker := time.NewTicker(consts.PollingInterval)
	defer ticker.Stop()

	// Wait for object of expected type to be deleted
	for {
		select {
		case <-chaosTestOverCtx.Done():
			t.Fatalf("timed out waiting for chaos scenario %s to finish", scenario)
		case <-ticker.C:
			obj, err := chaosClient.Resource(gvr).Namespace(namespace).Get(ctx, scenario, metav1.GetOptions{})
			if err != nil {
				if k8serrors.IsNotFound(err) {
					continue
				}
				t.Fatalf("failed to get chaos scenario %s: %v", scenario, err)
			}
			status, found, err := unstructured.NestedMap(obj.Object, "status")
			if err != nil || !found {
				continue
			}
			conditions, found, err := unstructured.NestedSlice(status, "conditions")
			if err != nil || !found {
				continue
			}
			for _, condition := range conditions {
				if condition == nil {
					continue
				}
				conditionMap, ok := condition.(map[string]interface{})
				if !ok {
					continue
				}
				if conditionMap["type"] == "AllRecovered" && conditionMap["status"] == "True" {
					return
				}
			}
		}
	}
}
