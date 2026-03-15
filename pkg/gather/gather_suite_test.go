package gather

import (
	"runtime"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGather(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gather Suite")
}

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
