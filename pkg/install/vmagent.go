package install

import (
	"context"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"

	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
)

// EnsureVMAgentRemoteWriteURL ensures that the specified VMAgent contains a remoteWrite
// entry with the provided URL. If no remoteWrite entries exist or the provided URL is
// not present, the function appends a remoteWrite entry with that URL and updates the
// VMAgent resource in Kubernetes.
//
// This helper is intended for use in end-to-end tests to guarantee that a VMAgent is
// configured to forward data to a particular remote endpoint (for example, a VMSingle
// instance used in overwatch tests).
//
// Parameters:
//   - ctx: context for the API requests and potential cancellation.
//   - t: terratest testing interface used for assertions and error reporting.
//   - vmclient: client for interacting with VictoriaMetrics Operator CRDs.
//   - kubeOpts: terratest kubectl options referring to the cluster and namespace (not used
//     directly for API calls here but kept for symmetry with other helpers).
//   - namespace: Kubernetes namespace where the VMAgent CR lives.
//   - vmAgentName: name of the VMAgent custom resource to inspect and potentially update.
//   - url: the remoteWrite URL that must be present in the VMAgent configuration.
func EnsureVMAgentRemoteWriteURL(ctx context.Context, t terratesting.TestingT, vmclient vmclient.Interface, kubeOpts *k8s.KubectlOptions, namespace, vmAgentName, url string) {
	// Get the VMAgent resource
	vmAgent, err := vmclient.OperatorV1beta1().VMAgents(namespace).Get(ctx, vmAgentName, metav1.GetOptions{})
	require.NoError(t, err)

	// Check if remoteWrite is configured and has at least one URL
	if vmAgent == nil || len(vmAgent.Spec.RemoteWrite) == 0 {
		t.Errorf("VMAgent %s in namespace %s does not have any remoteWrite configuration", vmAgentName, namespace)
		return
	}

	// Validate that at least one remoteWrite entry has a URL
	found := false
	for _, rw := range vmAgent.Spec.RemoteWrite {
		if rw.URL == url {
			found = true
			break
		}
	}
	if !found {
		// Get the fresh VMAgent resource version as it may have been updated by another test
		vmAgent, err := vmclient.OperatorV1beta1().VMAgents(namespace).Get(ctx, vmAgentName, metav1.GetOptions{})
		require.NoError(t, err)
		vmAgent.Spec.RemoteWrite = append(vmAgent.Spec.RemoteWrite, vmv1beta1.VMAgentRemoteWriteSpec{
			URL: url,
		})
		_, err = vmclient.OperatorV1beta1().VMAgents(namespace).Update(ctx, vmAgent, metav1.UpdateOptions{})
		require.NoError(t, err)
		WaitForVMAgentToBeOperational(ctx, t, kubeOpts, namespace, vmclient)
	}
}

// WaitForVMAgentToBeOperational watches the VMAgent custom resource in the given
// namespace and blocks until the agent reports an operational update status.
//
// The function uses a watch on VMAgent objects and a bounded timeout derived from
// consts.ResourceWaitTimeout. It returns by calling test assertions on the provided
// terratest testing interface if an error occurs during the wait.
//
// Parameters:
//   - ctx: parent context used for the watch and timeout propagation.
//   - t: terratest testing interface used for assertions and failing the test on errors.
//   - kubeOpts: terratest KubectlOptions pointing at the cluster/namespace (not used by the
//     watch but included for consistency with other helpers).
//   - namespace: the Kubernetes namespace where the VMAgent CR is located.
//   - vmclient: client for interacting with VictoriaMetrics Operator CRDs.
func WaitForVMAgentToBeOperational(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient vmclient.Interface) {
	watchInterface, err := vmclient.OperatorV1beta1().VMAgents(namespace).Watch(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	defer watchInterface.Stop()

	timeBoundContext, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
	defer cancel()

	_, err = watchtools.UntilWithoutRetry(timeBoundContext, watchInterface, func(event watch.Event) (bool, error) {
		obj := event.Object
		vmAgent := obj.(*vmv1beta1.VMAgent)
		if vmAgent.Status.UpdateStatus == vmv1beta1.UpdateStatusOperational {
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err)
}
