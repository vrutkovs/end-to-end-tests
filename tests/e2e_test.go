package end_to_end_tests_test

import (
	"testing"
	"time"

	terratesting "github.com/gruntwork-io/terratest/modules/testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	pollingInterval     = 5 * time.Second
	pollingTimeout      = 10 * time.Minute
	resourceWaitTimeout = 1 * time.Minute
)

var (
	retries = int(resourceWaitTimeout.Seconds() / pollingInterval.Seconds())
)

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
	return "[TerraTest+Ginkgo]"
}
