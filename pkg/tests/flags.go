package tests

import (
	"flag"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

var (
	reportLocation string
	envK8SDistro   string
)

func init() {
	flag.StringVar(&reportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&envK8SDistro, "env-k8s-distro", "kind", "Kube distro name")
}

func Init() {
	if !flag.Parsed() {
		flag.Parse()
	}
	consts.SetReportLocation(reportLocation)
	consts.SetEnvK8SDistro(envK8SDistro)
}
