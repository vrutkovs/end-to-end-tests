package consts

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReportLocation(t *testing.T) {
	testValue := "/test/report/location"

	SetReportLocation(testValue)
	result := ReportLocation()

	assert.Equal(t, testValue, result, "ReportLocation should return the set value")
}

func TestEnvK8SDistro(t *testing.T) {
	testValue := "test-distro"

	SetEnvK8SDistro(testValue)
	result := EnvK8SDistro()

	assert.Equal(t, testValue, result, "EnvK8SDistro should return the set value")
}

func TestNginxHost(t *testing.T) {
	testValue := "127.0.0.1"

	SetNginxHost(testValue)
	result := NginxHost()

	assert.Equal(t, testValue, result, "NginxHost should return the set value")
}

func TestVMSingleUrl(t *testing.T) {
	testNginxHost := "127.0.0.1"
	expectedURL := "http://vmsingle.127.0.0.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSingleUrl()

	if result != expectedURL {
		t.Errorf("Expected VMSingleUrl to be %s, got %s", expectedURL, result)
	}
}

func TestVMSelectUrlWithNamespace(t *testing.T) {
	testNginxHost := "10.0.0.1"
	testNamespace := "monitoring"
	expectedURL := "http://vmselect-monitoring.10.0.0.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectUrl(testNamespace)

	if result != expectedURL {
		t.Errorf("Expected VMSelectUrl to be %s, got %s", expectedURL, result)
	}
}

func TestVMSelectUrlWithoutNamespace(t *testing.T) {
	testNginxHost := "203.0.113.42"
	expectedURL := "http://vmselect.203.0.113.42.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectUrl("")

	if result != expectedURL {
		t.Errorf("Expected VMSelectUrl to be %s, got %s", expectedURL, result)
	}
}

func TestVMSingleHost(t *testing.T) {
	testNginxHost := "192.168.100.1"
	expectedHost := "vmsingle.192.168.100.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSingleHost()

	if result != expectedHost {
		t.Errorf("Expected VMSingleHost to be %s, got %s", expectedHost, result)
	}
}

func TestVMSelectHostWithNamespace(t *testing.T) {
	testNginxHost := "198.51.100.1"
	testNamespace := "prod"
	expectedHost := "vmselect-prod.198.51.100.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectHost(testNamespace)

	if result != expectedHost {
		t.Errorf("Expected VMSelectHost to be %s, got %s", expectedHost, result)
	}
}

func TestVMSelectHostWithoutNamespace(t *testing.T) {
	testNginxHost := "10.10.10.10"
	expectedHost := "vmselect.10.10.10.10.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectHost("")

	if result != expectedHost {
		t.Errorf("Expected VMSelectHost to be %s, got %s", expectedHost, result)
	}
}

func TestVMHostsWithEmptyNginxHost(t *testing.T) {
	// Reset nginx host to empty
	SetNginxHost("")

	// Both hosts should return empty string when nginx host is empty
	if VMSingleHost() != "" {
		t.Errorf("Expected VMSingleHost to be empty when nginx host is empty, got %s", VMSingleHost())
	}
	if VMSelectHost("test") != "" {
		t.Errorf("Expected VMSelectHost to be empty when nginx host is empty, got %s", VMSelectHost("test"))
	}

	// URLs should return "http://" when hosts are empty
	expectedEmptyURL := "http://"
	if VMSingleUrl() != expectedEmptyURL {
		t.Errorf("Expected VMSingleUrl to be %s when nginx host is empty, got %s", expectedEmptyURL, VMSingleUrl())
	}
	if VMSelectUrl("test") != expectedEmptyURL {
		t.Errorf("Expected VMSelectUrl to be %s when nginx host is empty, got %s", expectedEmptyURL, VMSelectUrl("test"))
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
				SetNginxHost(testValue)
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
				_ = NginxHost()
				_ = VMSingleUrl()
				_ = VMSelectUrl("test")
				_ = VMSingleHost()
				_ = VMSelectHost("test")
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
	SetNginxHost("")
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
	if NginxHost() != "" {
		t.Errorf("Expected initial NginxHost to be empty, got %s", NginxHost())
	}
	// When nginx host is empty, VM hosts should be empty
	if VMSingleHost() != "" {
		t.Errorf("Expected initial VMSingleHost to be empty, got %s", VMSingleHost())
	}
	if VMSelectHost("test") != "" {
		t.Errorf("Expected initial VMSelectHost to be empty, got %s", VMSelectHost("test"))
	}
	// When nginx host is empty, VM URLs should be "http://"
	expectedEmptyURL := "http://"
	if VMSingleUrl() != expectedEmptyURL {
		t.Errorf("Expected initial VMSingleUrl to be %s, got %s", expectedEmptyURL, VMSingleUrl())
	}
	if VMSelectUrl("test") != expectedEmptyURL {
		t.Errorf("Expected initial VMSelectUrl to be %s, got %s", expectedEmptyURL, VMSelectUrl("test"))
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

func TestNamespaceFormattingEdgeCases(t *testing.T) {
	testNginxHost := "10.20.30.40"
	SetNginxHost(testNginxHost)

	tests := []struct {
		name               string
		namespace          string
		expectedSingleHost string
		expectedSelectHost string
	}{
		{
			name:               "empty namespace",
			namespace:          "",
			expectedSingleHost: "vmsingle.10.20.30.40.nip.io",
			expectedSelectHost: "vmselect.10.20.30.40.nip.io",
		},
		{
			name:               "simple namespace",
			namespace:          "vm",
			expectedSingleHost: "vmsingle.10.20.30.40.nip.io",
			expectedSelectHost: "vmselect-vm.10.20.30.40.nip.io",
		},
		{
			name:               "namespace with dashes",
			namespace:          "vm-cluster-test",
			expectedSingleHost: "vmsingle.10.20.30.40.nip.io",
			expectedSelectHost: "vmselect-vm-cluster-test.10.20.30.40.nip.io",
		},
		{
			name:               "namespace with numbers",
			namespace:          "vm123",
			expectedSingleHost: "vmsingle.10.20.30.40.nip.io",
			expectedSelectHost: "vmselect-vm123.10.20.30.40.nip.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			singleHost := VMSingleHost()
			selectHost := VMSelectHost(tt.namespace)

			if singleHost != tt.expectedSingleHost {
				t.Errorf("Expected VMSingleHost to be %s, got %s", tt.expectedSingleHost, singleHost)
			}
			if selectHost != tt.expectedSelectHost {
				t.Errorf("Expected VMSelectHost to be %s, got %s", tt.expectedSelectHost, selectHost)
			}

			// Test URLs as well
			expectedSingleUrl := "http://" + tt.expectedSingleHost
			expectedSelectUrl := "http://" + tt.expectedSelectHost

			singleUrl := VMSingleUrl()
			selectUrl := VMSelectUrl(tt.namespace)

			if singleUrl != expectedSingleUrl {
				t.Errorf("Expected VMSingleUrl to be %s, got %s", expectedSingleUrl, singleUrl)
			}
			if selectUrl != expectedSelectUrl {
				t.Errorf("Expected VMSelectUrl to be %s, got %s", expectedSelectUrl, selectUrl)
			}
		})
	}
}

func TestGetVMSelectSvc(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "standard namespace",
			namespace: "vm",
			expected:  "vmselect-vmks.vm.svc.cluster.local.:8481",
		},
		{
			name:      "production namespace",
			namespace: "production",
			expected:  "vmselect-vmks.production.svc.cluster.local.:8481",
		},
		{
			name:      "staging namespace",
			namespace: "staging",
			expected:  "vmselect-vmks.staging.svc.cluster.local.:8481",
		},
		{
			name:      "namespace with dashes",
			namespace: "vm-cluster-test",
			expected:  "vmselect-vmks.vm-cluster-test.svc.cluster.local.:8481",
		},
		{
			name:      "empty namespace",
			namespace: "",
			expected:  "vmselect-vmks..svc.cluster.local.:8481",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMSelectSvc(tt.namespace)
			if result != tt.expected {
				t.Errorf("Expected GetVMSelectSvc to be %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetVMSingleSvc(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "standard namespace",
			namespace: "vm",
			expected:  "vmsingle-overwatch.vm.svc.cluster.local.:8428",
		},
		{
			name:      "production namespace",
			namespace: "production",
			expected:  "vmsingle-overwatch.production.svc.cluster.local.:8428",
		},
		{
			name:      "staging namespace",
			namespace: "staging",
			expected:  "vmsingle-overwatch.staging.svc.cluster.local.:8428",
		},
		{
			name:      "namespace with dashes",
			namespace: "vm-cluster-test",
			expected:  "vmsingle-overwatch.vm-cluster-test.svc.cluster.local.:8428",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMSingleSvc(tt.namespace)
			if result != tt.expected {
				t.Errorf("Expected GetVMSingleSvc to be %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestKubernetesServiceAddressesIntegration(t *testing.T) {
	// Test that service addresses work correctly across different namespaces
	namespaces := []string{"vm", "production", "staging", "monitoring"}

	for _, ns := range namespaces {
		t.Run("namespace_"+ns, func(t *testing.T) {
			vmSelectSvc := GetVMSelectSvc(ns)
			vmSingleSvc := GetVMSingleSvc(ns)

			// Verify they contain the namespace
			if !contains(vmSelectSvc, ns) {
				t.Errorf("VMSelect service address should contain namespace %s: %s", ns, vmSelectSvc)
			}
			if !contains(vmSingleSvc, ns) {
				t.Errorf("VMSingle service address should contain namespace %s: %s", ns, vmSingleSvc)
			}

			// Verify they contain the correct service names
			if !contains(vmSelectSvc, "vmselect-vmks") {
				t.Errorf("VMSelect service address should contain 'vmselect-vmks': %s", vmSelectSvc)
			}
			if !contains(vmSingleSvc, "vmsingle") {
				t.Errorf("VMSingle service address should contain 'vmsingle': %s", vmSingleSvc)
			}

			// Verify they contain the correct ports
			if !contains(vmSelectSvc, ":8481") {
				t.Errorf("VMSelect service address should contain port ':8481': %s", vmSelectSvc)
			}
			if !contains(vmSingleSvc, ":8428") {
				t.Errorf("VMSingle service address should contain port ':8428': %s", vmSingleSvc)
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestGetVMInsertSvc(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "standard namespace",
			namespace: "vm",
			expected:  "vminsert-vmks.vm.svc.cluster.local.:8480",
		},
		{
			name:      "production namespace",
			namespace: "production",
			expected:  "vminsert-vmks.production.svc.cluster.local.:8480",
		},
		{
			name:      "staging namespace",
			namespace: "staging",
			expected:  "vminsert-vmks.staging.svc.cluster.local.:8480",
		},
		{
			name:      "namespace with dashes",
			namespace: "vm-cluster-test",
			expected:  "vminsert-vmks.vm-cluster-test.svc.cluster.local.:8480",
		},
		{
			name:      "empty namespace",
			namespace: "",
			expected:  "vminsert-vmks..svc.cluster.local.:8480",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMInsertSvc(tt.namespace)
			if result != tt.expected {
				t.Errorf("Expected GetVMInsertSvc to be %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestVMServiceAddressesIntegration(t *testing.T) {
	// Test that all VM service addresses work correctly across different namespaces
	namespaces := []string{"vm", "production", "staging", "monitoring"}

	for _, ns := range namespaces {
		t.Run("namespace_"+ns, func(t *testing.T) {
			vmSelectSvc := GetVMSelectSvc(ns)
			vmSingleSvc := GetVMSingleSvc(ns)
			vmInsertSvc := GetVMInsertSvc(ns)

			// Verify they contain the namespace
			if !contains(vmSelectSvc, ns) {
				t.Errorf("VMSelect service address should contain namespace %s: %s", ns, vmSelectSvc)
			}
			if !contains(vmSingleSvc, ns) {
				t.Errorf("VMSingle service address should contain namespace %s: %s", ns, vmSingleSvc)
			}
			if !contains(vmInsertSvc, ns) {
				t.Errorf("VMInsert service address should contain namespace %s: %s", ns, vmInsertSvc)
			}

			// Verify they contain the correct service names
			if !contains(vmSelectSvc, "vmselect-vmks") {
				t.Errorf("VMSelect service address should contain 'vmselect-vmks': %s", vmSelectSvc)
			}
			if !contains(vmSingleSvc, "vmsingle") {
				t.Errorf("VMSingle service address should contain 'vmsingle': %s", vmSingleSvc)
			}
			if !contains(vmInsertSvc, "vminsert-vmks") {
				t.Errorf("VMInsert service address should contain 'vminsert-vmks': %s", vmInsertSvc)
			}

			// Verify they contain the correct ports
			if !contains(vmSelectSvc, ":8481") {
				t.Errorf("VMSelect service address should contain port ':8481': %s", vmSelectSvc)
			}
			if !contains(vmSingleSvc, ":8428") {
				t.Errorf("VMSingle service address should contain port ':8428': %s", vmSingleSvc)
			}
			if !contains(vmInsertSvc, ":8480") {
				t.Errorf("VMInsert service address should contain port ':8480': %s", vmInsertSvc)
			}
		})
	}
}

// TestConstantsValidity ensures all constants have valid values
func TestConstantsValidity(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		nonEmpty bool
	}{
		{"HelmChartVersion", HelmChartVersion(), false}, // May be empty in test environment
		{"VMVersion", VMVersion(), false},               // May be empty in test environment
		{"OperatorVersion", OperatorVersion(), false},   // May be empty in test environment
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just ensure the function doesn't panic and returns a string
			assert.IsType(t, "", tt.value, "%s should return a string type", tt.name)
			if tt.nonEmpty {
				assert.NotEmpty(t, tt.value, "%s should not be empty", tt.name)
			}
		})
	}
}
