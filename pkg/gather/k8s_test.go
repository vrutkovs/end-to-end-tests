package gather

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

type mockTestingT struct {
	failed bool
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) { m.failed = true }
func (m *mockTestingT) FailNow() {
	m.failed = true
	runtime.Goexit()
}
func (m *mockTestingT) Helper()                   {}
func (m *mockTestingT) Name() string              { return "mock" }
func (m *mockTestingT) Error(args ...interface{}) { m.failed = true }
func (m *mockTestingT) Fatal(args ...interface{}) {
	m.failed = true
	runtime.Goexit()
}
func (m *mockTestingT) Fatalf(format string, args ...interface{}) {
	m.failed = true
	runtime.Goexit()
}
func (m *mockTestingT) Log(args ...interface{})                 {}
func (m *mockTestingT) Logf(format string, args ...interface{}) {}
func (m *mockTestingT) Skip(args ...interface{})                {}
func (m *mockTestingT) SkipNow() {
	runtime.Goexit()
}
func (m *mockTestingT) Skipf(format string, args ...interface{}) {}
func (m *mockTestingT) Skipped() bool                            { return false }
func (m *mockTestingT) Fail()                                    { m.failed = true }
func (m *mockTestingT) Failed() bool                             { return m.failed }

func TestK8sAfterAll_EmptyLicense(t *testing.T) {
	gomega.RegisterTestingT(t)
	ctx := context.Background()
	mockT := &mockTestingT{}
	kubeOpts := &k8s.KubectlOptions{}

	// Ensure LicenseFile is empty so it skips cluster calls
	originalLicense := consts.LicenseFile()
	consts.SetLicenseFile("")
	defer consts.SetLicenseFile(originalLicense)

	// We use a short timeout because it will try to run commands that likely fail locally
	K8sAfterAll(ctx, mockT, kubeOpts, 10*time.Millisecond)

	// It should log some errors but not panic or fail the test
	assert.False(t, mockT.failed, "Expected K8sAfterAll not to fail when LicenseFile is empty")
}
