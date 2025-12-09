package consts

import "time"

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
	ReportLocation string
	EnvK8SDistro   string

	VMSingleUrl string
	VMSelectUrl string

	VMSingleHost string
	VMSelectHost string

	HelmChartVersion string
	VMVersion        string
	OperatorVersion  string
)
