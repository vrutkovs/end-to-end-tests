package tests

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func init() {
	flag.StringVar(&consts.ReportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&consts.EnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	vmSingleURL, err := url.Parse(consts.VMSingleUrl)
	if err != nil {
		panic(fmt.Errorf("failed to parse VMSingle URL: %w", err))
	}
	consts.VMSingleHost = vmSingleURL.Host

	vmSelectURL, err := url.Parse(consts.VMSelectUrl)
	if err != nil {
		panic(fmt.Errorf("failed to parse VMSelect URL: %w", err))
	}
	consts.VMSelectHost = vmSelectURL.Host
}
