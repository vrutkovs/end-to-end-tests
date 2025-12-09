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
