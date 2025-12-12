package install

import (
	"context"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"
)

func DiscoverIngressHost(ctx context.Context, t terratesting.TestingT) {
	kubeOpts := k8s.NewKubectlOptions("", "", "ingress-nginx")

	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "ingress-nginx-controller", consts.Retries, consts.PollingInterval)

	var nginxHost string

	// For kind environments, use localhost immediately
	if consts.EnvK8SDistro() == "kind" {
		logger.Default.Logf(t, "Kind environment detected, using localhost")
		nginxHost = "127.0.0.1"
	} else {
		// For non-kind environments, watch the service until LoadBalancer.Ingress is set
		nginxHost = waitForLoadBalancerIngress(ctx, t, kubeOpts)
	}

	logger.Default.Logf(t, "nginxHost: %s", nginxHost)

	// Set the discovered host in consts
	consts.SetNginxHost(nginxHost)
}

func waitForLoadBalancerIngress(ctx context.Context, t terratesting.TestingT, kubeOpts *k8s.KubectlOptions) string {
	logger.Default.Logf(t, "Waiting for ingress-nginx-controller service to have LoadBalancer.Ingress set...")

	// Create Kubernetes client from kubeOpts
	clientset, err := k8s.GetKubernetesClientFromOptionsE(t, kubeOpts)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %v", err)
		return ""
	}

	// Create a context with timeout
	watchCtx, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
	defer cancel()

	// First, check if the service already has LoadBalancer ingress
	svc, err := clientset.CoreV1().Services("ingress-nginx").Get(watchCtx, "ingress-nginx-controller", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("Failed to get ingress-nginx-controller service: %v", err)
		return ""
	}

	// Check if LoadBalancer ingress IP is already available
	if host := extractIngressHost(svc); host != "" {
		logger.Default.Logf(t, "LoadBalancer IP already available: %s", host)
		return host
	}

	// Set up field selector to watch only the specific service
	fieldSelector := fields.OneTermEqualSelector("metadata.name", "ingress-nginx-controller").String()

	// Create a watch for the service
	watcher, err := clientset.CoreV1().Services("ingress-nginx").Watch(watchCtx, metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
		return ""
	}
	defer watcher.Stop()

	// Define the condition function
	conditionFunc := func(event watch.Event) (bool, error) {
		switch event.Type {
		case watch.Modified, watch.Added:
			svc, ok := event.Object.(*corev1.Service)
			if !ok {
				return false, nil
			}
			// Check if LoadBalancer ingress IP is available
			if host := extractIngressHost(svc); host != "" {
				return true, nil
			}
			return false, nil
		default:
			return false, nil
		}
	}

	// Use watchtools.UntilWithoutRetry to watch for the condition
	event, err := watchtools.UntilWithoutRetry(watchCtx, watcher, conditionFunc)
	if err != nil {
		t.Fatalf("Failed to watch for LoadBalancer ingress: %v", err)
		return ""
	}

	// Extract the host from the final event
	if svc, ok := event.Object.(*corev1.Service); ok {
		if host := extractIngressHost(svc); host != "" {
			return host
		}
	}

	t.Fatalf("Failed to extract ingress host from final watch event")
	return ""
}

func extractIngressHost(svc *corev1.Service) string {
	if len(svc.Status.LoadBalancer.Ingress) > 0 {
		ingress := svc.Status.LoadBalancer.Ingress[0]

		// Only use IP address
		if ingress.IP != "" {
			return ingress.IP
		}
	}
	return ""
}
