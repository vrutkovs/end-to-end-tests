package tests

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	"github.com/Moon1706/ginkgo2allure/pkg/convert"
	fmngr "github.com/Moon1706/ginkgo2allure/pkg/convert/file_manager"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/parser"
	. "github.com/onsi/ginkgo/v2" //nolint
	"github.com/onsi/ginkgo/v2/types"
)

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

	reportPath, err := filepath.Abs(consts.ReportLocation())
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
		"kube-distro":      consts.EnvK8SDistro(),
		"helm-chart":       consts.HelmChartVersion(),
		"operator-version": consts.OperatorVersion(),
		"vm-version":       consts.VMVersion(),
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
