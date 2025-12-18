package install

import (
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2" //nolint

	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

// InstallVMGather provisions the VMGather deployment used by some end-to-end tests.
//
// Behavior:
//   - Ensures the `vmgather` namespace exists.
//   - Reads the VMGather manifest from the repository manifests.
//   - Replaces the placeholder host `vmgather.example.com` with the runtime value
//     provided by `consts.VMGatherHost()` so the ingress/host configuration matches
//     the test environment.
//   - Applies the modified manifest and waits for the `vmgather` deployment to become available.
//
// Parameters:
// - t: terratest testing interface used to perform kubectl operations and assertions.
func InstallVMGather(t terratesting.TestingT) {
	namespace := "vmgather"

	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	if _, err := k8s.GetNamespaceE(t, kubeOpts, namespace); err != nil {
		k8s.CreateNamespace(t, kubeOpts, namespace)
	}

	By("Install VMGather")
	// Copy vmsingle-ingress.yaml to temp file, update ingress host and apply it
	vmgather, err := os.ReadFile("../../manifests/vmgather.yaml")
	require.NoError(t, err)

	tempFile, err := os.CreateTemp("", "vmgather.yaml")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(tempFile.Name())
		require.NoError(t, err)
	}()

	// Extract host from consts.VMSingleUrl
	vmgather = []byte(strings.ReplaceAll(string(vmgather), "vmgather.example.com", consts.VMGatherHost()))

	_, err = tempFile.Write(vmgather)
	require.NoError(t, err)

	k8s.KubectlApply(t, kubeOpts, tempFile.Name())

	By("Wait for vmgather deployment to be available")
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmgather", consts.Retries, consts.PollingInterval)
}
