package consts

import (
	"log"
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
	log.Printf("Setting ReportLocation to: %s", val)
	reportLocation = val
}

func SetEnvK8SDistro(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting EnvK8SDistro to: %s", val)
	envK8SDistro = val
}

func SetVMSingleUrl(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting VMSingleUrl to: %s", val)
	vmSingleUrl = val
}

func SetVMSelectUrl(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting VMSelectUrl to: %s", val)
	vmSelectUrl = val
}

func SetVMSingleHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting VMSingleHost to: %s", val)
	vmSingleHost = val
}

func SetVMSelectHost(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting VMSelectHost to: %s", val)
	vmSelectHost = val
}

func SetHelmChartVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting HelmChartVersion to: %s", val)
	helmChartVersion = val
}

func SetVMVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting VMVersion to: %s", val)
	vmVersion = val
}

func SetOperatorVersion(val string) {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Setting OperatorVersion to: %s", val)
	operatorVersion = val
}

// Getters
func ReportLocation() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting ReportLocation: %s", reportLocation)
	return reportLocation
}

func EnvK8SDistro() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting EnvK8SDistro: %s", envK8SDistro)
	return envK8SDistro
}

func VMSingleUrl() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting VMSingleUrl: %s", vmSingleUrl)
	return vmSingleUrl
}

func VMSelectUrl() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting VMSelectUrl: %s", vmSelectUrl)
	return vmSelectUrl
}

func VMSingleHost() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting VMSingleHost: %s", vmSingleHost)
	return vmSingleHost
}

func VMSelectHost() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting VMSelectHost: %s", vmSelectHost)
	return vmSelectHost
}

func HelmChartVersion() string {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Getting HelmChartVersion: %s", helmChartVersion)
	return helmChartVersion
}

func VMVersion() string {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Getting VMVersion: %s", vmVersion)
	return vmVersion
}

func OperatorVersion() string {
	mu.Lock()
	defer mu.Unlock()
	log.Printf("Getting OperatorVersion: %s", operatorVersion)
	return operatorVersion
}
