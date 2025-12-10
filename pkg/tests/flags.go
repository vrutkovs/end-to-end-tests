package tests

import (
	"flag"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

var (
	reportLocation string
	envK8SDistro   string
	vmTag          string
)

func init() {
	flag.StringVar(&reportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&envK8SDistro, "env-k8s-distro", "kind", "Kube distro name")
	flag.StringVar(&vmTag, "vmtag", "", "VictoriaMetrics image tag to use for testing")
}

func Init() {
	if !flag.Parsed() {
		flag.Parse()
	}
	consts.SetReportLocation(reportLocation)
	consts.SetEnvK8SDistro(envK8SDistro)
	consts.SetVMTag(vmTag)
}

func GetVMTag() string {
	return vmTag
}
