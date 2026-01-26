package install

import (
	"context"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
)

// WaitForVMSingleToBeOperational watches a VMSingle custom resource until it reports an operational status.
//
// The function sets up a watch for VMSingle objects in the provided namespace and
// blocks until the VMSingle's Status.UpdateStatus becomes UpdateStatusOperational or
// the wait times out. It uses consts.ResourceWaitTimeout to bound the wait.
func WaitForVMSingleToBeOperational(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient vmclient.Interface) {
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
