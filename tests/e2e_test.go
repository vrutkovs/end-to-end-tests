package end_to_end_tests_test

import (
	"path/filepath"
	"testing"

	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	"github.com/Moon1706/ginkgo2allure/pkg/convert"
	fmngr "github.com/Moon1706/ginkgo2allure/pkg/convert/file_manager"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/parser"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

func TestSmokeTests(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.LabelFilter = "smoke"
	RunSpecs(t, "Smoke test Suite", suiteConfig, reporterConfig)
}

func TestLoadTestsTests(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	suiteConfig.LabelFilter = "load-test"
	RunSpecs(t, "Load test Suite", suiteConfig, reporterConfig)
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

	reportPath, err := filepath.Abs("../allure-results")
	if err != nil {
		panic(err)
	}
	fileManager := fmngr.NewFileManager(reportPath)

	errs := convert.PrintAllureReports(allureReports, fileManager)
	if len(errs) > 0 {
		panic(errs)
	}
})
