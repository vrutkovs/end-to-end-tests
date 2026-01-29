package consts

import (
	"fmt"
	"sync"
	"time"
)

const (
	// PollingInterval is the interval at which tests verify conditions (e.g. resource readiness).
	PollingInterval = 30 * time.Second
	// PollingTimeout defines the overall timeout for polling operations.
	PollingTimeout = 15 * time.Minute
	// ResourceWaitTimeout is the maximum duration to wait for Kubernetes resources to become available.
	ResourceWaitTimeout = 10 * time.Minute

	// K6JobPollingInterval is the interval for checking K6 job status.
	K6JobPollingInterval = 1 * time.Minute
	// K6JobMaxDuration is the maximum allowed duration for a K6 load test job.
	K6JobMaxDuration = 60 * time.Minute

	// ChaosTestMaxDuration is the maximum allowed duration for a Chaos Mesh scenario.
	ChaosTestMaxDuration = 30 * time.Minute
)

var (
	// Retries is the number of attempts to make based on ResourceWaitTimeout and PollingInterval.
	Retries = int(ResourceWaitTimeout.Seconds() / PollingInterval.Seconds())
	// K6Retries is the number of attempts for K6 jobs based on K6JobMaxDuration.
	K6Retries = int(K6JobMaxDuration.Seconds() / K6JobPollingInterval.Seconds())
)

var (
	mu sync.Mutex

	reportLocation string
	envK8SDistro   string

	nginxHost string

	helmChartVersion string
	vmVersion        string
	operatorVersion  string
)

// Setters

// SetReportLocation sets the path for test reports.
func SetReportLocation(val string) {
	mu.Lock()
	defer mu.Unlock()
	reportLocation = val
}

// SetEnvK8SDistro sets the Kubernetes distribution name (e.g., kind, gke).
func SetEnvK8SDistro(val string) {
	mu.Lock()
	defer mu.Unlock()
	envK8SDistro = val
}

// SetNginxHost sets the external hostname for Nginx ingress.
func SetNginxHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	nginxHost = val
}

// SetHelmChartVersion sets the detected Helm chart version.
func SetHelmChartVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	helmChartVersion = val
}

// SetVMVersion sets the detected VictoriaMetrics version.
func SetVMVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmVersion = val
}

// SetOperatorVersion sets the detected VictoriaMetrics Operator version.
func SetOperatorVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	operatorVersion = val
}

// SetVMTag sets the VictoriaMetrics image tag to use.
func SetVMTag(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmVersion = val
}

// Getters

// ReportLocation returns the configured report location.
func ReportLocation() string {
	mu.Lock()
	defer mu.Unlock()
	return reportLocation
}

// EnvK8SDistro returns the configured Kubernetes distribution.
func EnvK8SDistro() string {
	mu.Lock()
	defer mu.Unlock()
	return envK8SDistro
}

// NginxHost returns the configured Nginx host.
func NginxHost() string {
	mu.Lock()
	defer mu.Unlock()
	return nginxHost
}

// VMSingleUrl constructs the URL for the VMSingle instance.
func VMSingleUrl() string {
	return fmt.Sprintf("http://%s", VMSingleHost())
}

// VMSelectUrl constructs the URL for the VMSelect instance in the given namespace.
func VMSelectUrl(namespace string) string {
	return fmt.Sprintf("http://%s", VMSelectHost(namespace))
}

// VMSingleHost returns the hostname for VMSingle.
func VMSingleHost() string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	return fmt.Sprintf("vmsingle.%s.nip.io", host)
}

// VMSingleNamespacedHost returns the hostname for VMSingle in the given namespace.
func VMSingleNamespacedHost(namespace string) string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	return fmt.Sprintf("vmsingle-%s.%s.nip.io", namespace, host)
}

// VMAgentNamespacedHost returns the hostname for VMAgent in the given namespace.
func VMAgentNamespacedHost(namespace string) string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	return fmt.Sprintf("vmagent-%s.%s.nip.io", namespace, host)
}

// VMSelectHost returns the hostname for VMSelect in the given namespace.
func VMSelectHost(namespace string) string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	if namespace == "" {
		return fmt.Sprintf("vmselect.%s.nip.io", host)
	}
	return fmt.Sprintf("vmselect-%s.%s.nip.io", namespace, host)
}

// VMInsertHost returns the hostname for VMInsert in the given namespace.
func VMInsertHost(namespace string) string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	if namespace == "" {
		return fmt.Sprintf("vminsert.%s.nip.io", host)
	}
	return fmt.Sprintf("vminsert-%s.%s.nip.io", namespace, host)
}

// VMGatherHost returns the hostname for VMGather.
func VMGatherHost() string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	return fmt.Sprintf("vmgather.%s.nip.io", host)
}

// Kubernetes service address functions

// GetVMSelectSvc returns the internal Kubernetes service address for VMSelect.
func GetVMSelectSvc(releaseName, namespace string) string {
	return fmt.Sprintf("vmselect-%s.%s.svc.cluster.local:8481", releaseName, namespace)
}

// GetVMSingleSvc returns the internal Kubernetes service address for VMSingle.
func GetVMSingleSvc(releaseName, namespace string) string {
	return fmt.Sprintf("vmsingle-%s.%s.svc.cluster.local:8428", releaseName, namespace)
}

// GetVMInsertSvc returns the internal Kubernetes service address for VMInsert.
func GetVMInsertSvc(releaseName, namespace string) string {
	return fmt.Sprintf("vminsert-%s.%s.svc.cluster.local:8480", releaseName, namespace)
}

// HelmChartVersion returns the stored Helm chart version.
func HelmChartVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return helmChartVersion
}

// VMVersion returns the stored VictoriaMetrics version.
func VMVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return vmVersion
}

// OperatorVersion returns the stored Operator version.
func OperatorVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return operatorVersion
}
