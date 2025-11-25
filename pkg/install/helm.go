package install

import (
	"context"
	"fmt"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"

	. "github.com/onsi/ginkgo/v2" // nolint

	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func InstallWithHelm(ctx context.Context, helmChart, valuesFile string, t terratesting.TestingT, namespace string, releaseName string) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	helmOpts := &helm.Options{
		KubectlOptions: kubeOpts,
		ValuesFiles:    []string{valuesFile},
		ExtraArgs: map[string][]string{
			"upgrade": {"--create-namespace", "--wait"},
		},
	}
	vmclient := GetVMClient(t, kubeOpts)

	By(fmt.Sprintf("Install %s chart", helmChart))
	helm.Upgrade(t, helmOpts, helmChart, releaseName)
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmagent-vmks", consts.Retries, consts.PollingInterval)
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmalert-vmks", consts.Retries, consts.PollingInterval)
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vminsert-vmks", consts.Retries, consts.PollingInterval)

	By("Install VMSingle overwatch instance")
	k8s.KubectlApply(t, kubeOpts, "../../manifests/overwatch/vmsingle.yaml")
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmsingle-overwatch", consts.Retries, consts.PollingInterval)

	By("Reconfigure VMAgent to send data to VMSingle")
	k8s.KubectlApply(t, kubeOpts, "../../manifests/overwatch/vmagent.yaml")

	WaitForVMClusterToBeOperational(ctx, t, kubeOpts, namespace, vmclient)
	WaitForVMSingleToBeOperational(ctx, t, kubeOpts, namespace, vmclient)
}

func GetVMClient(t terratesting.TestingT, kubeOpts *k8s.KubectlOptions) *vmclient.Clientset {
	kubeConfigPath, err := kubeOpts.GetConfigPath(t)
	require.NoError(t, err)
	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}, &clientcmd.ConfigOverrides{})
	restConfig, err := clientConfig.ClientConfig()
	require.NoError(t, err)
	vmclient := vmclient.NewForConfigOrDie(restConfig)
	require.NoError(t, err)
	return vmclient
}

func WaitForVMSingleToBeOperational(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient *vmclient.Clientset) {
	By("Wait for overwatch VMSingle to become operational")
	watchInterface, err := vmclient.OperatorV1beta1().VMSingles(namespace).Watch(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	defer watchInterface.Stop()

	timeBoundContext, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
	defer cancel()

	_, err = watchtools.UntilWithoutRetry(timeBoundContext, watchInterface, func(event watch.Event) (bool, error) {
		obj := event.Object
		vmSingle := obj.(*vmv1beta1.VMSingle)
		if vmSingle.Status.UpdateStatus == vmv1beta1.UpdateStatusOperational {
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err)
}

func WaitForVMClusterToBeOperational(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient *vmclient.Clientset) {
	By("Wait for VMCluster object to become operational")
	watchInterface, err := vmclient.OperatorV1beta1().VMClusters(namespace).Watch(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	defer watchInterface.Stop()

	timeBoundContext, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
	defer cancel()

	_, err = watchtools.UntilWithoutRetry(timeBoundContext, watchInterface, func(event watch.Event) (bool, error) {
		obj := event.Object
		vmCluster := obj.(*vmv1beta1.VMCluster)
		if vmCluster.Status.UpdateStatus == vmv1beta1.UpdateStatusOperational {
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err)
}
