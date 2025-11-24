package consts

import "time"

const (
	PollingInterval     = 30 * time.Second
	PollingTimeout      = 10 * time.Minute
	ResourceWaitTimeout = 10 * time.Minute
)

var (
	Retries = int(ResourceWaitTimeout.Seconds() / PollingInterval.Seconds())
)
