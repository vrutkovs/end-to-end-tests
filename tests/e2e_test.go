package end_to_end_tests_test

import (
	"flag"
	"testing"
	"time"

	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	"github.com/Moon1706/ginkgo2allure/pkg/convert"
	fmngr "github.com/Moon1706/ginkgo2allure/pkg/convert/file_manager"
	"github.com/Moon1706/ginkgo2allure/pkg/convert/parser"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

const (
	pollingInterval     = 5 * time.Second
	pollingTimeout      = 10 * time.Minute
	resourceWaitTimeout = 1 * time.Minute
)

var (
	retries      = int(resourceWaitTimeout.Seconds() / pollingInterval.Seconds())
	vmClusterUrl string
)

func init() {
	flag.StringVar(&vmClusterUrl, "vmcluster", "", "VMCluster URL")
}

func TestEndToEndTests(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EndToEndTests Suite")
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

// Extends GinkgoTInterface to have #Name() method, that is compatible with testing.TestingT
func (mt *myTestingT) Name() string {
	return "[TerraTest]"
}

var _ = ReportAfterSuite("allure report", func(report types.Report) {
	allureReports, err := convert.GinkgoToAllureReport([]types.Report{report}, parser.NewDefaultParser, parser.Config{})
	if err != nil {
		panic(err)
	}

	fileManager := fmngr.NewFileManager("./allure-results")
	errs := convert.PrintAllureReports(allureReports, fileManager)
	if len(errs) > 0 {
		panic(errs)
	}
})
