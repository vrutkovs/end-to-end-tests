package install

import (
	"context"
	"fmt"
	"os"

	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	k6v1alpha1 "github.com/grafana/k6-operator/api/v1alpha1"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func InstallK6(ctx context.Context, t terratesting.TestingT, namespace string) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	k8s.KubectlApply(t, kubeOpts, "../../manifests/k6-operator/bundle.yaml")
	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "k6-operator-controller-manager", consts.Retries, consts.PollingInterval)
}

func RunK6Scenario(ctx context.Context, t terratesting.TestingT, namespace, scenario, vmselectURL string, parallelism int) error {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)

	scenarioPath := fmt.Sprintf("../../manifests/load-tests/%s.js", scenario)
	scenarioContent, err := os.ReadFile(scenarioPath)
	if err != nil {
		return fmt.Errorf("failed to read scenario file: %w", err)
	}

	// Create a configmap with a script
	configMap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenario,
			Namespace: namespace,
		},
		Data: map[string]string{
			"script.js": string(scenarioContent),
		},
	}
	yamlConfigMap, err := yaml.Marshal(configMap)
	if err != nil {
		return fmt.Errorf("failed to marshal configMap: %w", err)
	}
	k8s.KubectlApplyFromString(t, kubeOpts, string(yamlConfigMap))

	// Create TestRun CR
	testRun := &k6v1alpha1.TestRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "TestRun",
			APIVersion: "k6.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      scenario,
			Namespace: namespace,
		},
		Spec: k6v1alpha1.TestRunSpec{
			Script: k6v1alpha1.K6Script{
				ConfigMap: k6v1alpha1.K6Configmap{
					Name: scenario,
					File: "script.js",
				},
			},
			Parallelism: int32(parallelism),
		},
	}
	yamlTestRun, err := yaml.Marshal(testRun)
	if err != nil {
		return fmt.Errorf("failed to marshal testRun: %w", err)
	}
	k8s.KubectlApplyFromString(t, kubeOpts, string(yamlTestRun))

	k8s.WaitUntilJobSucceed(t, kubeOpts, fmt.Sprintf("%s-initializer", scenario), consts.Retries, consts.PollingInterval)
	k8s.WaitUntilJobSucceed(t, kubeOpts, fmt.Sprintf("%s-starter", scenario), consts.Retries, consts.PollingInterval)
	return nil
}

func WaitForK6JobsToComplete(ctx context.Context, t terratesting.TestingT, namespace, scenario string, parallelism int) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)

	for idx := 0; idx < parallelism; idx++ {
		k8s.WaitUntilJobSucceed(t, kubeOpts, fmt.Sprintf("%s-%d", scenario, idx+1), consts.K6Retries, consts.K6JobPollingInterval)
	}
}
