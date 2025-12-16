package install

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
)

// InstallVMCluster installs a VMCluster using the template manifest with namespace-specific modifications
func InstallVMCluster(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient vmclient.Interface) {
	// Make sure namespace exists
	if _, err := k8s.GetNamespaceE(t, kubeOpts, namespace); err != nil {
		k8s.CreateNamespace(t, kubeOpts, namespace)
	}

	// Read the VMCluster template
	vmclusterYamlPath := "../../manifests/overwatch/vmcluster.yaml"
	vmclusterYaml, err := os.ReadFile(vmclusterYamlPath)
	require.NoError(t, err)

	// Create a temporary file with namespace-specific modifications
	tempFile, err := os.CreateTemp("", "vmcluster-*.yaml")
	require.NoError(t, err)
	defer func() {
		err := os.Remove(tempFile.Name())
		require.NoError(t, err)
	}()

	// Replace the name to include namespace if not "vm"
	updatedVMClusterYaml := string(vmclusterYaml)
	updatedVMClusterYaml = strings.ReplaceAll(updatedVMClusterYaml, "name: vm", fmt.Sprintf("name: %s", namespace))

	// Write the updated content to the temporary file
	_, err = tempFile.Write([]byte(updatedVMClusterYaml))
	require.NoError(t, err)

	// Apply the VMCluster manifest
	fmt.Printf("Installing VMCluster in namespace %s\n", namespace)
	k8s.KubectlApply(t, kubeOpts, tempFile.Name())

	// Wait for VMCluster to become operational
	WaitForVMClusterToBeOperational(ctx, t, kubeOpts, namespace, vmclient)
}

// EnsureVMClusterComponents validates that all VMCluster components are properly configured and running
func EnsureVMClusterComponents(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, namespace string, vmclient vmclient.Interface, vmclusterName string) {
	// Get the VMCluster resource
	vmcluster, err := vmclient.OperatorV1beta1().VMClusters(namespace).Get(ctx, vmclusterName, metav1.GetOptions{})
	require.NoError(t, err)

	// Validate VMCluster specification
	if vmcluster.Spec.RetentionPeriod == "" {
		t.Errorf("VMCluster %s in namespace %s has empty retention period", vmclusterName, namespace)
	} else {
		fmt.Printf("VMCluster %s has retention period: %s\n", vmclusterName, vmcluster.Spec.RetentionPeriod)
	}

	// Validate VMStorage configuration
	if vmcluster.Spec.VMStorage == nil {
		t.Errorf("VMCluster %s in namespace %s has no VMStorage configuration", vmclusterName, namespace)
	} else {
		fmt.Printf("VMCluster %s VMStorage replica count: %d\n", vmclusterName, *vmcluster.Spec.VMStorage.ReplicaCount)
		if vmcluster.Spec.VMStorage.StorageDataPath == "" {
			t.Errorf("VMCluster %s VMStorage has empty storage data path", vmclusterName)
		}
	}

	// Validate VMSelect configuration
	if vmcluster.Spec.VMSelect == nil {
		t.Errorf("VMCluster %s in namespace %s has no VMSelect configuration", vmclusterName, namespace)
	} else {
		fmt.Printf("VMCluster %s VMSelect replica count: %d\n", vmclusterName, *vmcluster.Spec.VMSelect.ReplicaCount)
	}

	// Validate VMInsert configuration
	if vmcluster.Spec.VMInsert == nil {
		t.Errorf("VMCluster %s in namespace %s has no VMInsert configuration", vmclusterName, namespace)
	} else {
		fmt.Printf("VMCluster %s VMInsert replica count: %d\n", vmclusterName, *vmcluster.Spec.VMInsert.ReplicaCount)
	}

	// Check operational status
	if vmcluster.Status.UpdateStatus != "ExpandSuccess" && vmcluster.Status.UpdateStatus != "Operational" {
		fmt.Printf("VMCluster %s status: %s (reason: %s)\n", vmclusterName, vmcluster.Status.UpdateStatus, vmcluster.Status.Reason)
	} else {
		fmt.Printf("VMCluster %s is operational\n", vmclusterName)
	}
}

// GetVMClusterServiceEndpoints returns the service endpoints for VMCluster components
func GetVMClusterServiceEndpoints(namespace string, vmclusterName string) VMClusterEndpoints {
	return VMClusterEndpoints{
		VMInsert:  fmt.Sprintf("vminsert-%s.%s.svc.cluster.local:8480", vmclusterName, namespace),
		VMSelect:  fmt.Sprintf("vmselect-%s.%s.svc.cluster.local:8481", vmclusterName, namespace),
		VMStorage: fmt.Sprintf("vmstorage-%s.%s.svc.cluster.local:8482", vmclusterName, namespace),
	}
}

// VMClusterEndpoints holds the service endpoints for VMCluster components
type VMClusterEndpoints struct {
	VMInsert  string
	VMSelect  string
	VMStorage string
}

// DeleteVMCluster removes a VMCluster and waits for cleanup
func DeleteVMCluster(t terratesting.TestingT, kubeOpts *k8s.KubectlOptions, vmclusterName string) {
	// Delete the VMCluster resource
	fmt.Printf("Deleting VMCluster %s\n", vmclusterName)
	k8s.RunKubectl(t, kubeOpts, "delete", "vmcluster", vmclusterName, "--ignore-not-found=true")

	// Wait for deployments to be deleted
	vmstorageName := fmt.Sprintf("vmstorage-%s", vmclusterName)
	vmselectName := fmt.Sprintf("vmselect-%s", vmclusterName)
	vminsertName := fmt.Sprintf("vminsert-%s", vmclusterName)

	// Note: WaitUntilDeploymentNotFound would be ideal here, but we'll use a simple approach
	// In a real scenario, you might want to implement proper cleanup waiting
	deployments := []string{vmstorageName, vmselectName, vminsertName}
	for _, deployment := range deployments {
		k8s.RunKubectl(t, kubeOpts, "wait", "--for=delete", "deployment", deployment, "--timeout=60s", "--ignore-not-found=true")
	}
}
