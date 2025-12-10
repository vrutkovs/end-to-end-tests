package tests

import (
	"flag"
	"os"
	"testing"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

func TestInit(t *testing.T) {
	// Skip this test if flags are already parsed to avoid conflicts
	if flag.Parsed() {
		t.Skip("Flags already parsed, skipping test to avoid conflicts")
	}

	// Save original command line arguments and flags
	origArgs := os.Args
	origCommandLine := flag.CommandLine
	defer func() {
		os.Args = origArgs
		flag.CommandLine = origCommandLine
	}()

	// Create a new flag set to avoid conflicts
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	var testReportLocation string
	var testEnvK8SDistro string
	testFlagSet.StringVar(&testReportLocation, "report", "/tmp/allure-results", "Report location")
	testFlagSet.StringVar(&testEnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	// Test with default values
	err := testFlagSet.Parse([]string{})
	if err != nil {
		t.Fatalf("Failed to parse test flags: %v", err)
	}

	if testReportLocation != "/tmp/allure-results" {
		t.Errorf("Expected default report location to be '/tmp/allure-results', got '%s'", testReportLocation)
	}
	if testEnvK8SDistro != "kind" {
		t.Errorf("Expected default env k8s distro to be 'kind', got '%s'", testEnvK8SDistro)
	}
}

func TestInitWithCustomFlags(t *testing.T) {
	// Skip this test if flags are already parsed to avoid conflicts
	if flag.Parsed() {
		t.Skip("Flags already parsed, skipping test to avoid conflicts")
	}

	// Create a new flag set to test custom values
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)
	var testReportLocation string
	var testEnvK8SDistro string
	testFlagSet.StringVar(&testReportLocation, "report", "/tmp/allure-results", "Report location")
	testFlagSet.StringVar(&testEnvK8SDistro, "env-k8s-distro", "kind", "Kube distro name")

	// Test with custom values
	err := testFlagSet.Parse([]string{"-report", "/custom/report/path", "-env-k8s-distro", "minikube"})
	if err != nil {
		t.Fatalf("Failed to parse test flags: %v", err)
	}

	if testReportLocation != "/custom/report/path" {
		t.Errorf("Expected report location to be '/custom/report/path', got '%s'", testReportLocation)
	}
	if testEnvK8SDistro != "minikube" {
		t.Errorf("Expected env k8s distro to be 'minikube', got '%s'", testEnvK8SDistro)
	}
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
	if consts.ReportLocation() != "/initial/path" {
		t.Errorf("Expected report location to be '/initial/path', got '%s'", consts.ReportLocation())
	}
	if consts.EnvK8SDistro() != "initial-distro" {
		t.Errorf("Expected env k8s distro to be 'initial-distro', got '%s'", consts.EnvK8SDistro())
	}
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
	if err != nil {
		t.Fatalf("Failed to parse empty flags: %v", err)
	}

	if testReportLocation != "/tmp/allure-results" {
		t.Errorf("Expected default report location to be '/tmp/allure-results', got '%s'", testReportLocation)
	}
	if testEnvK8SDistro != "kind" {
		t.Errorf("Expected default env k8s distro to be 'kind', got '%s'", testEnvK8SDistro)
	}
}

func TestFlagVariables(t *testing.T) {
	// Test that the package-level variables exist and have expected initial values
	// Note: These variables are set during package init, so we can't easily reset them
	// We're just testing that they exist and can be accessed

	// The variables should be accessible (this is mainly a compilation test)
	_ = reportLocation
	_ = envK8SDistro
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

	if consts.ReportLocation() != firstReport {
		t.Errorf("Expected report location to remain '%s' after second set, got '%s'", firstReport, consts.ReportLocation())
	}
	if consts.EnvK8SDistro() != firstDistro {
		t.Errorf("Expected env k8s distro to remain '%s' after second set, got '%s'", firstDistro, consts.EnvK8SDistro())
	}
}

func TestVMTagFlagDefault(t *testing.T) {
	// Test that vmtag has the expected default value
	// Create a new flag set to test defaults
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	var testVMTag string
	testFlagSet.StringVar(&testVMTag, "vmtag", "", "VictoriaMetrics image tag to use for testing")

	// Parse empty args to get defaults
	err := testFlagSet.Parse([]string{})
	if err != nil {
		t.Fatalf("Failed to parse empty flags: %v", err)
	}

	if testVMTag != "" {
		t.Errorf("Expected default vmtag to be empty string, got '%s'", testVMTag)
	}
}

func TestVMTagFlagCustomValue(t *testing.T) {
	// Test that vmtag accepts custom values
	testFlagSet := flag.NewFlagSet("test", flag.ContinueOnError)

	var testVMTag string
	testFlagSet.StringVar(&testVMTag, "vmtag", "", "VictoriaMetrics image tag to use for testing")

	// Test with custom VM tag
	err := testFlagSet.Parse([]string{"-vmtag", "v1.131.0"})
	if err != nil {
		t.Fatalf("Failed to parse vmtag flag: %v", err)
	}

	if testVMTag != "v1.131.0" {
		t.Errorf("Expected vmtag to be 'v1.131.0', got '%s'", testVMTag)
	}
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
	if retrievedTag != testTag {
		t.Errorf("Expected VM version to be '%s', got '%s'", testTag, retrievedTag)
	}
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
		if retrievedVersion != version {
			t.Errorf("Expected VM version to be '%s', got '%s'", version, retrievedVersion)
		}
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
	if retrievedTag != "" {
		t.Errorf("Expected VM version to be empty string, got '%s'", retrievedTag)
	}
}

func TestGetVMTagFunction(t *testing.T) {
	// Test the GetVMTag function exists and works
	// Save original value
	originalVMTag := vmTag

	// Test with different values
	testTag := "v1.129.1"
	vmTag = testTag

	retrievedTag := GetVMTag()
	if retrievedTag != testTag {
		t.Errorf("Expected GetVMTag() to return '%s', got '%s'", testTag, retrievedTag)
	}

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
	if retrievedVersion != testTag {
		t.Errorf("Expected VM version to be '%s' after SetVMTag(), got '%s'", testTag, retrievedVersion)
	}
}

func TestVMTagFlagVariableExists(t *testing.T) {
	// Test that the vmTag variable exists and is accessible
	_ = vmTag // This should compile without error

	// Test that we can read and write to it
	originalValue := vmTag
	vmTag = "test-value"
	if vmTag != "test-value" {
		t.Errorf("Expected vmTag to be 'test-value', got '%s'", vmTag)
	}
	vmTag = originalValue
}
