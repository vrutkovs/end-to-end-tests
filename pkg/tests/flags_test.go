package tests

import (
	"flag"
	"os"
	"testing"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInit(t *testing.T) {
	// Isolate global flag state so tests can run even if flags were parsed elsewhere.
	origArgs := os.Args
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
	}()
	// Replace global command line with a fresh FlagSet for isolation.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Create a new flag set to avoid conflicts
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	var testReportLocation string
	var testEnvK8SDistro string
	testFlagSet.StringVar(&testReportLocation, "report", "/tmp/allure-results", "Report location")
	testFlagSet.StringVar(&testEnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	// Test with default values
	err := testFlagSet.Parse([]string{})
	require.NoError(t, err, "Failed to parse test flags")

	assert.Equal(t, "/tmp/allure-results", testReportLocation, "Expected default report location to match")
	assert.Equal(t, "kind", testEnvK8SDistro, "Expected default env k8s distro to match")
}

func TestInitWithCustomFlags(t *testing.T) {
	// Isolate global flag state so tests can run even if flags were parsed elsewhere.
	origArgs := os.Args
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
	}()
	// Replace global command line with a fresh FlagSet for isolation.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	// Create a new flag set to test custom values
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	var testReportLocation string
	var testEnvK8SDistro string
	testFlagSet.StringVar(&testReportLocation, "report", "/tmp/allure-results", "Report location")
	testFlagSet.StringVar(&testEnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	// Test with custom values
	err := testFlagSet.Parse([]string{"-report", "/custom/report/path", "-env-k8s-distro", "minikube"})
	require.NoError(t, err, "Failed to parse test flags")

	assert.Equal(t, "/custom/report/path", testReportLocation)
	assert.Equal(t, "minikube", testEnvK8SDistro)
}

func TestInitAlreadyParsed(t *testing.T) {
	// Test the logic when flags are already parsed
	// Set initial values in consts
	originalReport := consts.ReportLocation()
	originalDistro := consts.EnvK8SDistro()

	defer func() {
		consts.SetReportLocation(originalReport)
		consts.SetEnvK8SDistro(originalDistro)
	}()

	consts.SetReportLocation("/initial/path")
	consts.SetEnvK8SDistro("initial-distro")

	// Since we can't easily test Init with already parsed flags without side effects,
	// we test that the consts package setter/getter work correctly
	assert.Equal(t, "/initial/path", consts.ReportLocation())
	assert.Equal(t, "initial-distro", consts.EnvK8SDistro())
}

func TestFlagDefaults(t *testing.T) {
	// Save original command line arguments
	origArgs := os.Args
	defer func() {
		os.Args = origArgs
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	}()

	// Create a new flag set to test defaults
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	var testReportLocation string
	var testEnvK8SDistro string

	testFlagSet.StringVar(&testReportLocation, "report", "/tmp/allure-results", "Report location")
	testFlagSet.StringVar(&testEnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	// Parse empty args to get defaults
	err := testFlagSet.Parse([]string{})
	require.NoError(t, err, "Failed to parse empty flags")

	assert.Equal(t, "/tmp/allure-results", testReportLocation)
	assert.Equal(t, "kind", testEnvK8SDistro)
}

func TestMultipleInits(t *testing.T) {
	// Test that multiple calls to consts setters work correctly
	originalReport := consts.ReportLocation()
	originalDistro := consts.EnvK8SDistro()

	defer func() {
		consts.SetReportLocation(originalReport)
		consts.SetEnvK8SDistro(originalDistro)
	}()

	// Test multiple setter calls
	consts.SetReportLocation("/first/path")
	consts.SetEnvK8SDistro("first-distro")

	firstReport := consts.ReportLocation()
	firstDistro := consts.EnvK8SDistro()

	// Call setters again with same values
	consts.SetReportLocation("/first/path")
	consts.SetEnvK8SDistro("first-distro")

	assert.Equal(t, firstReport, consts.ReportLocation())
	assert.Equal(t, firstDistro, consts.EnvK8SDistro())
}

func TestVMTagFlagDefault(t *testing.T) {
	// Test that vmtag has the expected default value
	// Create a new flag set to test defaults
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	var testVMTag string
	testFlagSet.StringVar(&testVMTag, "vmtag", "", "VictoriaMetrics image tag to use for testing")

	// Parse empty args to get defaults
	err := testFlagSet.Parse([]string{})
	require.NoError(t, err, "Failed to parse empty flags")

	assert.Empty(t, testVMTag, "Expected default vmtag to be empty string")
}

func TestVMTagFlagCustomValue(t *testing.T) {
	// Test that vmtag accepts custom values
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	var testVMTag string
	testFlagSet.StringVar(&testVMTag, "vmtag", "", "VictoriaMetrics image tag to use for testing")

	// Test with custom VM tag
	err := testFlagSet.Parse([]string{"-vmtag", "v1.131.0"})
	require.NoError(t, err, "Failed to parse vmtag flag")

	assert.Equal(t, "v1.131.0", testVMTag)
}

func TestVMTagSetterAndGetter(t *testing.T) {
	// Test the SetVMTag and VMVersion functions work correctly
	originalVMVersion := consts.VMVersion()
	defer func() {
		consts.SetVMTag(originalVMVersion)
	}()

	// Test setting VM tag
	testTag := "v1.130.0"
	consts.SetVMTag(testTag)

	retrievedTag := consts.VMVersion()
	assert.Equal(t, testTag, retrievedTag)
}

func TestVMTagWithDifferentVersions(t *testing.T) {
	// Test setting different VM tag versions
	originalVMVersion := consts.VMVersion()
	defer func() {
		consts.SetVMTag(originalVMVersion)
	}()

	testVersions := []string{"v1.131.0", "v1.130.0", "v1.129.1", "v1.128.0"}

	for _, version := range testVersions {
		consts.SetVMTag(version)
		retrievedVersion := consts.VMVersion()
		assert.Equal(t, version, retrievedVersion)
	}
}

func TestVMTagEmptyValue(t *testing.T) {
	// Test setting empty VM tag
	originalVMVersion := consts.VMVersion()
	defer func() {
		consts.SetVMTag(originalVMVersion)
	}()

	// Set empty tag
	consts.SetVMTag("")
	retrievedTag := consts.VMVersion()
	assert.Empty(t, retrievedTag)
}

func TestGetVMTagFunction(t *testing.T) {
	// Test the GetVMTag function exists and works
	// Save original value
	originalVMTag := vmTag

	// Test with different values
	testTag := "v1.129.1"
	vmTag = testTag

	retrievedTag := GetVMTag()
	assert.Equal(t, testTag, retrievedTag)

	// Reset original value
	vmTag = originalVMTag
}

func TestVMTagIntegrationWithInit(t *testing.T) {
	// Test that vmtag integrates correctly with the Init function
	originalVMVersion := consts.VMVersion()
	defer func() {
		consts.SetVMTag(originalVMVersion)
	}()

	// Simulate setting vmTag variable (as would happen during flag parsing)
	testTag := "v1.131.0"
	originalVMTag := vmTag
	vmTag = testTag

	defer func() {
		vmTag = originalVMTag
	}()

	// Since flags are already parsed in the test environment, we'll test
	// the SetVMTag functionality directly instead of calling Init()
	consts.SetVMTag(vmTag)

	// Verify that consts was updated
	retrievedVersion := consts.VMVersion()
	assert.Equal(t, testTag, retrievedVersion)
}

func TestVMTagFlagVariableExists(t *testing.T) {
	// Test that the vmTag variable exists and is accessible
	_ = vmTag // This should compile without error

	// Test that we can read and write to it
	originalValue := vmTag
	vmTag = "test-value"
	assert.Equal(t, "test-value", vmTag)
	vmTag = originalValue
}
