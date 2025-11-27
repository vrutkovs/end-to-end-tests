package tests

import (
	"flag"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	"github.com/Moon1706/ginkgo2allure/pkg/convert"
	fmngr "github.com/Moon1706/ginkgo2allure/pkg/convert/file_manager"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/parser"
	. "github.com/onsi/ginkgo/v2" //nolint
	"github.com/onsi/ginkgo/v2/types"
)

var (
	reportLocation string
	envK8SDistro   string
)

func init() {
	flag.StringVar(&reportLocation, "report", "/tmp/allure-results", "Report location")
	flag.StringVar(&envK8SDistro, "env-k8s-distro", "", "Kube distro name")
}

// GetT returns a testing.T compatible object that can be used in terratesting.RunE2ETests
func GetT() terratesting.TestingT {
	return &myTestingT{
		GinkgoT(),
	}
}

type myTestingT struct {
	GinkgoTInterface
}

func (mt *myTestingT) Name() string {
	return ""
}

var _ = ReportAfterSuite("allure report", func(report types.Report) {
	allureReports, err := convert.GinkgoToAllureReport([]types.Report{report}, parser.NewDefaultParser, parser.Config{})
	if err != nil {
		panic(err)
	}

	reportPath, err := filepath.Abs(reportLocation)
	if err != nil {
		panic(err)
	}

	if err := writeEnvironmentProperties(reportPath); err != nil {
		panic(err)
	}

	fileManager := fmngr.NewFileManager(reportPath)

	errs := convert.PrintAllureReports(allureReports, fileManager)
	if len(errs) > 0 {
		panic(errs)
	}
})

func writeEnvironmentProperties(reportPath string) error {
	envFilePath := filepath.Join(reportPath, "environment.properties")
	if err := os.MkdirAll(filepath.Dir(envFilePath), 0755); err != nil {
		return err
	}

	props := map[string]string{
		"kube-distro":      envK8SDistro,
		"helm-chart":       install.HelmChartVersion,
		"operator-version": install.OperatorVersion,
		"vm-version":       install.VMVersion,
	}

	return os.WriteFile(envFilePath, environmentPropertiesContent(props), 0644)
}

func environmentPropertiesContent(props map[string]string) []byte {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var builder strings.Builder
	for _, k := range keys {
		builder.WriteString(k)
		builder.WriteString("=")
		builder.WriteString(props[k])
		builder.WriteString("\n")
	}

	return []byte(builder.String())
}
