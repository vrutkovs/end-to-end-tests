package tests

import (
	"flag"
	"fmt"
	"net/url"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func init() {
	flag.StringVar(&consts.ReportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&consts.EnvK8SDistro, "env-k8s-distro", "", "Kube distro name")
	flag.StringVar(&consts.VMSingleUrl, "vmsingle-url", "http://vmsingle.34.116.133.143.nip.io", "VMSingle ingress")
	flag.StringVar(&consts.VMSelectUrl, "vmselect-url", "http://vmselect.34.116.133.143.nip.io", "VMSelect ingress")

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
