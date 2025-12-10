package consts

import (
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

	vmSingleUrl string
	vmSelectUrl string

	vmSingleHost string
	vmSelectHost string

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

func SetVMSingleUrl(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmSingleUrl = val
}

func SetVMSelectUrl(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmSelectUrl = val
}

func SetVMSingleHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmSingleHost = val
}

func SetVMSelectHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	vmSelectHost = val
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

func VMSingleUrl() string {
	mu.Lock()
	defer mu.Unlock()
	return vmSingleUrl
}

func VMSelectUrl() string {
	mu.Lock()
	defer mu.Unlock()
	return vmSelectUrl
}

func VMSingleHost() string {
	mu.Lock()
	defer mu.Unlock()
	return vmSingleHost
}

func VMSelectHost() string {
	mu.Lock()
	defer mu.Unlock()
	return vmSelectHost
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
