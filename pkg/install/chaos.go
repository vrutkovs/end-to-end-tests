package install

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2" // nolint
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "chaos-controller-manager", consts.Retries, consts.PollingInterval)
}

func RunChaosScenario(ctx context.Context, t terratesting.TestingT, scenarioFolder, scenario, chaosType string) error {
	namespace := "vm"
	gvr := schema.GroupVersionResource{Group: "chaos-mesh.org", Version: "v1alpha1", Resource: chaosType}

	chaosTestOverCtx, cancel := context.WithTimeout(ctx, consts.ChaosTestMaxDuration)
	defer cancel()

	// Create dynamic client
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	kubeConfigPath, err := kubeOpts.GetConfigPath(t)
	require.NoError(t, err)
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	require.NoError(t, err)
	dynClient, err := dynamic.NewForConfig(restConfig)
	require.NoError(t, err)

	// Apply manifest, this starts the chaos scenario
	manifestPath := fmt.Sprintf("../../manifests/chaos-tests/%s/%s.yaml", scenarioFolder, scenario)
	k8s.KubectlApply(t, kubeOpts, manifestPath)

	ticker := time.NewTicker(consts.ChaosTestMaxDuration)
	defer ticker.Stop()

	// Wait for object of expected type to be deleted
	for {
		select {
		case <-chaosTestOverCtx.Done():
			return fmt.Errorf("timed out waiting for chaos scenario %s to finish", scenario)
		case <-ticker.C:
			_, err := dynClient.Resource(gvr).Namespace(namespace).Get(ctx, filepath.Base(scenario), metav1.GetOptions{})
			if k8serrors.IsNotFound(err) {
				return nil
			}
		}
	}
}
