package consts

import (
	"fmt"
	"sync"
	"time"
)

const (
	// PollingInterval is the interval at which tests verify conditions (e.g. resource readiness).
	PollingInterval = 5 * time.Second
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

	// HTTPClientTimeout is the default timeout for HTTP clients used in tests.
	HTTPClientTimeout = 10 * time.Second

	// DataPropagationDelay is the time to wait for data to propagate through the system.
	DataPropagationDelay = 30 * time.Second

	// AggregationWaitTime is the time to wait for streaming aggregation to complete.
	AggregationWaitTime = 1 * time.Minute
)

// Common namespace constants used across tests.
const (
	// DefaultVMNamespace is the default namespace for VictoriaMetrics deployments.
	DefaultVMNamespace = "monitoring"

	// OverwatchNamespace is the namespace for the overwatch monitoring stack.
	OverwatchNamespace = "overwatch"

	// K6OperatorNamespace is the namespace for the k6 operator.
	K6OperatorNamespace = "k6-operator-system"

	// K6TestsNamespace is the namespace for running k6 tests.
	K6TestsNamespace = "k6-tests"

	// BenchmarkNamespace is the namespace for prometheus benchmark.
	BenchmarkNamespace = "vm-benchmark"

	// ChaosMeshNamespace is the namespace for chaos mesh.
	ChaosMeshNamespace = "chaos-mesh"
)

// Common release and resource names used across tests.
const (
	// DefaultReleaseName is the default Helm release name for VM k8s stack.
	DefaultReleaseName = "vmks"

	// DefaultVMClusterName is the default name for VMCluster resources.
	DefaultVMClusterName = "vm"

	// ChaosMeshReleaseName is the Helm release name for chaos mesh.
	ChaosMeshReleaseName = "chaos-mesh"
)

// Helm chart references.
const (
	// VMK8sStackChart is the Helm chart for VictoriaMetrics k8s stack.
	VMK8sStackChart = "vm/victoria-metrics-k8s-stack"

	// VMDistributedChart is the Helm chart for VictoriaMetrics distributed deployment.
	VMDistributedChart = "vm/victoria-metrics-distributed"

	// ChaosMeshChart is the Helm chart for Chaos Mesh.
	ChaosMeshChart = "chaos-mesh/chaos-mesh"
)

// Values file paths (relative to test directories).
const (
	// ManifestsRoot is the shared base path for manifest files used across the
	// test helpers.
	ManifestsRoot = "../../manifests"

	// Overwatch manifests
	OverwatchVMSingleYaml    = ManifestsRoot + "/overwatch/vmsingle.yaml"
	OverwatchVMAgentYaml     = ManifestsRoot + "/overwatch/vmagent.yaml"
	OverwatchVMSingleIngress = ManifestsRoot + "/overwatch/vmsingle-ingress.yaml"

	// SmokeValuesFile is the values file for smoke tests.
	SmokeValuesFile = ManifestsRoot + "/smoke.yaml"

	// DistributedValuesFile is the values file for distributed chart tests.
	DistributedValuesFile = ManifestsRoot + "/distributed.yaml"

	// ChaosMeshValuesFile is the values file for chaos mesh.
	ChaosMeshValuesFile = ManifestsRoot + "/chaos-mesh-operator/values.yaml"
)

// Common error messages.
const (
	// ErrNoDataReturned is the error message when a query returns no data.
	ErrNoDataReturned = "no data returned"
)

// URL path patterns for VictoriaMetrics endpoints.
const (
	// PrometheusPathSuffix is the suffix for Prometheus-compatible endpoints.
	PrometheusPathSuffix = "/prometheus"

	// TenantInsertPathFormat is the format for tenant-specific insert URLs.
	// Arguments: tenant ID
	TenantInsertPathFormat = "/insert/%d/prometheus/api/v1/write"

	// TenantSelectPathFormat is the format for tenant-specific select URLs.
	// Arguments: tenant ID
	TenantSelectPathFormat = "/select/%d/prometheus"

	// MultitenantInsertPath is the path for multitenant insert endpoint.
	MultitenantInsertPath = "/insert/multitenant/prometheus/api/v1/write"

	// MultitenantSelectPath is the path for multitenant select endpoint.
	MultitenantSelectPath = "/select/multitenant/prometheus"

	// RemoteWritePath is the path for remote write API.
	RemoteWritePath = "/api/v1/write"
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
