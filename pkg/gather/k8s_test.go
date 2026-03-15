package gather

import (
	"context"
	"time"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("K8s Gather", func() {
	var (
		ctx context.Context
	)

	BeforeEach(func() {
		ctx = context.Background()
	})

	Context("K8sAfterAll", func() {
		It("should execute without crashing when LicenseFile is empty", func() {
			mockT := &mockTestingT{}
			kubeOpts := &k8s.KubectlOptions{}

			// Ensure LicenseFile is empty so it skips cluster calls
			originalLicense := consts.LicenseFile()
			consts.SetLicenseFile("")
			defer consts.SetLicenseFile(originalLicense)

			// We use a short timeout because it will try to run commands that likely fail locally
			K8sAfterAll(ctx, mockT, kubeOpts, 1*time.Second)

			// It should log some errors but not panic or fail the test
			Expect(mockT.failed).To(BeFalse())
		})
	})
})
