package tests

import (
	"flag"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func init() {
	flag.StringVar(&consts.ReportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&consts.EnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")
}
