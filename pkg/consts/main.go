package consts

import (
	"fmt"
	"sync"
	"time"
)

const (
	PollingInterval     = 30 * time.Second
	PollingTimeout      = 10 * time.Minute
	ResourceWaitTimeout = 10 * time.Minute

	K6JobPollingInterval = 1 * time.Minute
	K6JobMaxDuration     = 60 * time.Minute

	ChaosTestMaxDuration = 30 * time.Minute
)

var (
	Retries   = int(ResourceWaitTimeout.Seconds() / PollingInterval.Seconds())
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
func SetReportLocation(val string) {
	mu.Lock()
	defer mu.Unlock()
	reportLocation = val
}

func SetEnvK8SDistro(val string) {
	mu.Lock()
	defer mu.Unlock()
	envK8SDistro = val
}

func SetNginxHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	nginxHost = val
}

func SetHelmChartVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	helmChartVersion = val
}

func SetVMVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmVersion = val
}

func SetOperatorVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	operatorVersion = val
}

func SetVMTag(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmVersion = val
}

// Getters
func ReportLocation() string {
	mu.Lock()
	defer mu.Unlock()
	return reportLocation
}

func EnvK8SDistro() string {
	mu.Lock()
	defer mu.Unlock()
	return envK8SDistro
}

func NginxHost() string {
	mu.Lock()
	defer mu.Unlock()
	return nginxHost
}

func VMSingleUrl() string {
	return fmt.Sprintf("http://%s", VMSingleHost())
}

func VMSelectUrl(namespace string) string {
	return fmt.Sprintf("http://%s", VMSelectHost(namespace))
}

func VMSingleHost() string {
	mu.Lock()
	host := nginxHost
	mu.Unlock()
	if host == "" {
		return ""
	}
	return fmt.Sprintf("vmsingle.%s.nip.io", host)
}

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
func GetVMSelectSvc(namespace string) string {
	return fmt.Sprintf("vmselect-vmks.%s.svc.cluster.local.:8481", namespace)
}

func GetVMSingleSvc(namespace string) string {
	return fmt.Sprintf("vmsingle-overwatch.%s.svc.cluster.local.:8428", namespace)
}

func GetVMInsertSvc(namespace string) string {
	return fmt.Sprintf("vminsert-vmks.%s.svc.cluster.local.:8480", namespace)
}

func HelmChartVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return helmChartVersion
}

func VMVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return vmVersion
}

func OperatorVersion() string {
	mu.Lock()
	defer mu.Unlock()
	return operatorVersion
}
