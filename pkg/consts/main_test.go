package consts

import (
	"sync"
	"testing"
)

func TestReportLocation(t *testing.T) {
	testValue := "/test/report/location"

	SetReportLocation(testValue)
	result := ReportLocation()

	if result != testValue {
		t.Errorf("Expected ReportLocation to be %s, got %s", testValue, result)
	}
}

func TestEnvK8SDistro(t *testing.T) {
	testValue := "test-distro"

	SetEnvK8SDistro(testValue)
	result := EnvK8SDistro()

	if result != testValue {
		t.Errorf("Expected EnvK8SDistro to be %s, got %s", testValue, result)
	}
}

func TestVMSingleUrl(t *testing.T) {
	testValue := "http://test-vm-single.example.com"

	SetVMSingleUrl(testValue)
	result := VMSingleUrl()

	if result != testValue {
		t.Errorf("Expected VMSingleUrl to be %s, got %s", testValue, result)
	}
}

func TestVMSelectUrl(t *testing.T) {
	testValue := "http://test-vm-select.example.com"

	SetVMSelectUrl(testValue)
	result := VMSelectUrl()

	if result != testValue {
		t.Errorf("Expected VMSelectUrl to be %s, got %s", testValue, result)
	}
}

func TestVMSingleHost(t *testing.T) {
	testValue := "test-vm-single.example.com"

	SetVMSingleHost(testValue)
	result := VMSingleHost()

	if result != testValue {
		t.Errorf("Expected VMSingleHost to be %s, got %s", testValue, result)
	}
}

func TestVMSelectHost(t *testing.T) {
	testValue := "test-vm-select.example.com"

	SetVMSelectHost(testValue)
	result := VMSelectHost()

	if result != testValue {
		t.Errorf("Expected VMSelectHost to be %s, got %s", testValue, result)
	}
}

func TestHelmChartVersion(t *testing.T) {
	testValue := "v1.2.3"

	SetHelmChartVersion(testValue)
	result := HelmChartVersion()

	if result != testValue {
		t.Errorf("Expected HelmChartVersion to be %s, got %s", testValue, result)
	}
}

func TestVMVersion(t *testing.T) {
	testValue := "v1.95.0"

	SetVMVersion(testValue)
	result := VMVersion()

	if result != testValue {
		t.Errorf("Expected VMVersion to be %s, got %s", testValue, result)
	}
}

func TestOperatorVersion(t *testing.T) {
	testValue := "v0.47.0"

	SetOperatorVersion(testValue)
	result := OperatorVersion()

	if result != testValue {
		t.Errorf("Expected OperatorVersion to be %s, got %s", testValue, result)
	}
}

func TestConcurrentAccess(t *testing.T) {
	// Test thread safety by running concurrent reads and writes
	const numGoroutines = 100
	const numOperations = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // One for setters, one for getters

	// Run setters concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				testValue := "test-value-" + string(rune(id)) + "-" + string(rune(j))
				SetReportLocation(testValue)
				SetEnvK8SDistro(testValue)
				SetVMSingleUrl(testValue)
				SetVMSelectUrl(testValue)
				SetVMSingleHost(testValue)
				SetVMSelectHost(testValue)
				SetHelmChartVersion(testValue)
				SetVMVersion(testValue)
				SetOperatorVersion(testValue)
			}
		}(i)
	}

	// Run getters concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = ReportLocation()
				_ = EnvK8SDistro()
				_ = VMSingleUrl()
				_ = VMSelectUrl()
				_ = VMSingleHost()
				_ = VMSelectHost()
				_ = HelmChartVersion()
				_ = VMVersion()
				_ = OperatorVersion()
			}
		}()
	}

	wg.Wait()
	// If we get here without race conditions or panics, the test passes
}

func TestInitialValues(t *testing.T) {
	// Reset all values by setting them to empty strings
	SetReportLocation("")
	SetEnvK8SDistro("")
	SetVMSingleUrl("")
	SetVMSelectUrl("")
	SetVMSingleHost("")
	SetVMSelectHost("")
	SetHelmChartVersion("")
	SetVMVersion("")
	SetOperatorVersion("")

	// Test that initial values are empty
	if ReportLocation() != "" {
		t.Errorf("Expected initial ReportLocation to be empty, got %s", ReportLocation())
	}
	if EnvK8SDistro() != "" {
		t.Errorf("Expected initial EnvK8SDistro to be empty, got %s", EnvK8SDistro())
	}
	if VMSingleUrl() != "" {
		t.Errorf("Expected initial VMSingleUrl to be empty, got %s", VMSingleUrl())
	}
	if VMSelectUrl() != "" {
		t.Errorf("Expected initial VMSelectUrl to be empty, got %s", VMSelectUrl())
	}
	if VMSingleHost() != "" {
		t.Errorf("Expected initial VMSingleHost to be empty, got %s", VMSingleHost())
	}
	if VMSelectHost() != "" {
		t.Errorf("Expected initial VMSelectHost to be empty, got %s", VMSelectHost())
	}
	if HelmChartVersion() != "" {
		t.Errorf("Expected initial HelmChartVersion to be empty, got %s", HelmChartVersion())
	}
	if VMVersion() != "" {
		t.Errorf("Expected initial VMVersion to be empty, got %s", VMVersion())
	}
	if OperatorVersion() != "" {
		t.Errorf("Expected initial OperatorVersion to be empty, got %s", OperatorVersion())
	}
}

func TestSetVMTag(t *testing.T) {
	// Test that SetVMTag correctly sets the VM version
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	testTag := "v1.131.0"
	SetVMTag(testTag)
	result := VMVersion()

	if result != testTag {
		t.Errorf("Expected VMVersion to be %s after SetVMTag, got %s", testTag, result)
	}
}

func TestSetVMTagWithDifferentVersions(t *testing.T) {
	// Test setting various VM tag versions
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	testVersions := []string{
		"v1.131.0",
		"v1.130.0",
		"v1.129.1",
		"v1.128.0",
		"latest",
		"nightly",
		"",
	}

	for _, version := range testVersions {
		SetVMTag(version)
		result := VMVersion()
		if result != version {
			t.Errorf("Expected VMVersion to be '%s' after SetVMTag, got '%s'", version, result)
		}
	}
}

func TestSetVMTagEmptyString(t *testing.T) {
	// Test setting VM tag to empty string
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	SetVMTag("")
	result := VMVersion()

	if result != "" {
		t.Errorf("Expected VMVersion to be empty string after SetVMTag(''), got '%s'", result)
	}
}

func TestSetVMTagConcurrency(t *testing.T) {
	// Test thread safety of SetVMTag
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	const numGoroutines = 50
	const numOperations = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2)

	// Run SetVMTag concurrently
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				testTag := "v1.130." + string(rune('0'+id%10))
				SetVMTag(testTag)
			}
		}(i)
	}

	// Run VMVersion getter concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				_ = VMVersion()
			}
		}()
	}

	wg.Wait()
	// If we get here without race conditions or panics, the test passes
}

func TestSetVMTagIntegration(t *testing.T) {
	// Test integration between SetVMTag and other VM-related functions
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	// Test that SetVMTag works independently of SetVMVersion
	testTag := "v1.131.0"
	SetVMTag(testTag)

	// Verify it's set correctly
	if VMVersion() != testTag {
		t.Errorf("Expected VMVersion to be %s, got %s", testTag, VMVersion())
	}

	// Test that SetVMVersion still works after SetVMTag
	differentTag := "v1.130.0"
	SetVMVersion(differentTag)

	if VMVersion() != differentTag {
		t.Errorf("Expected VMVersion to be %s after SetVMVersion, got %s", differentTag, VMVersion())
	}

	// Test that SetVMTag can override SetVMVersion
	finalTag := "v1.129.1"
	SetVMTag(finalTag)

	if VMVersion() != finalTag {
		t.Errorf("Expected VMVersion to be %s after final SetVMTag, got %s", finalTag, VMVersion())
	}
}
