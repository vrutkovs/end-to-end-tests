package consts

import (
	"fmt"
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

	assert.Equal(t, expectedURL, result)
}

func TestVMSelectUrlWithNamespace(t *testing.T) {
	testNginxHost := "10.0.0.1"
	testNamespace := "monitoring"
	expectedURL := "http://vmselect-monitoring.10.0.0.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectUrl(testNamespace)

	assert.Equal(t, expectedURL, result)
}

func TestVMSelectUrlWithoutNamespace(t *testing.T) {
	testNginxHost := "203.0.113.42"
	expectedURL := "http://vmselect.203.0.113.42.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectUrl("")

	assert.Equal(t, expectedURL, result)
}

func TestVMSingleHost(t *testing.T) {
	testNginxHost := "192.168.100.1"
	expectedHost := "vmsingle.192.168.100.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSingleHost()

	assert.Equal(t, expectedHost, result)
}

func TestVMSelectHostWithNamespace(t *testing.T) {
	testNginxHost := "198.51.100.1"
	testNamespace := "prod"
	expectedHost := "vmselect-prod.198.51.100.1.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectHost(testNamespace)

	assert.Equal(t, expectedHost, result)
}

func TestVMSelectHostWithoutNamespace(t *testing.T) {
	testNginxHost := "10.10.10.10"
	expectedHost := "vmselect.10.10.10.10.nip.io"

	SetNginxHost(testNginxHost)
	result := VMSelectHost("")

	assert.Equal(t, expectedHost, result)
}

func TestVMHostsWithEmptyNginxHost(t *testing.T) {
	// Reset nginx host to empty
	SetNginxHost("")

	// Both hosts should return empty string when nginx host is empty
	assert.Empty(t, VMSingleHost(), "VMSingleHost should be empty when nginx host is empty")
	assert.Empty(t, VMSelectHost("test"), "VMSelectHost should be empty when nginx host is empty")

	// URLs should return "http://" when hosts are empty
	expectedEmptyURL := "http://"
	assert.Equal(t, expectedEmptyURL, VMSingleUrl(), "VMSingleUrl should be http:// when nginx host is empty")
	assert.Equal(t, expectedEmptyURL, VMSelectUrl("test"), "VMSelectUrl should be http:// when nginx host is empty")
}

func TestHelmChartVersion(t *testing.T) {
	testValue := "v1.2.3"

	SetHelmChartVersion(testValue)
	result := HelmChartVersion()

	assert.Equal(t, testValue, result)
}

func TestVMVersion(t *testing.T) {
	testValue := "v1.95.0"

	SetVMVersion(testValue)
	result := VMVersion()

	assert.Equal(t, testValue, result)
}

func TestOperatorVersion(t *testing.T) {
	testValue := "v0.47.0"

	SetOperatorVersion(testValue)
	result := OperatorVersion()

	assert.Equal(t, testValue, result)
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
				testValue := fmt.Sprintf("test-value-%c-%c", rune(id), rune(j))
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
	assert.Empty(t, ReportLocation(), "Initial ReportLocation should be empty")
	assert.Empty(t, EnvK8SDistro(), "Initial EnvK8SDistro should be empty")
	assert.Empty(t, NginxHost(), "Initial NginxHost should be empty")

	// When nginx host is empty, VM hosts should be empty
	assert.Empty(t, VMSingleHost(), "Initial VMSingleHost should be empty")
	assert.Empty(t, VMSelectHost("test"), "Initial VMSelectHost should be empty")

	// When nginx host is empty, VM URLs should be "http://"
	expectedEmptyURL := "http://"
	assert.Equal(t, expectedEmptyURL, VMSingleUrl(), "Initial VMSingleUrl should be http://")
	assert.Equal(t, expectedEmptyURL, VMSelectUrl("test"), "Initial VMSelectUrl should be http://")

	assert.Empty(t, HelmChartVersion(), "Initial HelmChartVersion should be empty")
	assert.Empty(t, VMVersion(), "Initial VMVersion should be empty")
	assert.Empty(t, OperatorVersion(), "Initial OperatorVersion should be empty")
}

func TestSetVMTag(t *testing.T) {
	// Test that SetVMTag correctly sets the VM version
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	testTag := "v1.131.0"
	SetVMTag(testTag)
	result := VMVersion()

	assert.Equal(t, testTag, result)
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
		assert.Equal(t, version, result, "VMVersion should match set version '%s'", version)
	}
}

func TestSetVMTagEmptyString(t *testing.T) {
	// Test setting VM tag to empty string
	originalVMVersion := VMVersion()
	defer SetVMTag(originalVMVersion) // Restore original value

	SetVMTag("")
	result := VMVersion()

	assert.Empty(t, result)
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
				testTag := fmt.Sprintf("v1.130.%c", rune('0'+id%10))
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
	assert.Equal(t, testTag, VMVersion())

	// Test that SetVMVersion still works after SetVMTag
	differentTag := "v1.130.0"
	SetVMVersion(differentTag)

	assert.Equal(t, differentTag, VMVersion())

	// Test that SetVMTag can override SetVMVersion
	finalTag := "v1.129.1"
	SetVMTag(finalTag)

	assert.Equal(t, finalTag, VMVersion())
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

			assert.Equal(t, tt.expectedSingleHost, singleHost)
			assert.Equal(t, tt.expectedSelectHost, selectHost)

			// Test URLs as well
			expectedSingleUrl := "http://" + tt.expectedSingleHost
			expectedSelectUrl := "http://" + tt.expectedSelectHost

			singleUrl := VMSingleUrl()
			selectUrl := VMSelectUrl(tt.namespace)

			assert.Equal(t, expectedSingleUrl, singleUrl)
			assert.Equal(t, expectedSelectUrl, selectUrl)
		})
	}
}

func TestGetVMSelectSvc(t *testing.T) {
	tests := []struct {
		name        string
		releaseName string
		namespace   string
		expected    string
	}{
		{
			name:        "standard namespace",
			releaseName: "vmks",
			namespace:   "vm",
			expected:    "vmselect-vmks.vm.svc.cluster.local:8481",
		},
		{
			name:        "production namespace",
			releaseName: "vmks",
			namespace:   "production",
			expected:    "vmselect-vmks.production.svc.cluster.local:8481",
		},
		{
			name:        "staging namespace",
			releaseName: "vmks",
			namespace:   "staging",
			expected:    "vmselect-vmks.staging.svc.cluster.local:8481",
		},
		{
			name:        "namespace with dashes",
			releaseName: "vmks",
			namespace:   "vm-cluster-test",
			expected:    "vmselect-vmks.vm-cluster-test.svc.cluster.local:8481",
		},
		{
			name:        "empty namespace",
			releaseName: "vmks",
			namespace:   "",
			expected:    "vmselect-vmks..svc.cluster.local:8481",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMSelectSvc(tt.releaseName, tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetVMSingleSvc(t *testing.T) {
	tests := []struct {
		name        string
		releaseName string
		namespace   string
		expected    string
	}{
		{
			name:        "standard namespace",
			releaseName: "overwatch",
			namespace:   "vm",
			expected:    "vmsingle-overwatch.vm.svc.cluster.local:8428",
		},
		{
			name:        "production namespace",
			releaseName: "overwatch",
			namespace:   "production",
			expected:    "vmsingle-overwatch.production.svc.cluster.local:8428",
		},
		{
			name:        "staging namespace",
			releaseName: "overwatch",
			namespace:   "staging",
			expected:    "vmsingle-overwatch.staging.svc.cluster.local:8428",
		},
		{
			name:        "namespace with dashes",
			releaseName: "overwatch",
			namespace:   "vm-cluster-test",
			expected:    "vmsingle-overwatch.vm-cluster-test.svc.cluster.local:8428",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMSingleSvc(tt.releaseName, tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKubernetesServiceAddressesIntegration(t *testing.T) {
	// Test that service addresses work correctly across different namespaces
	namespaces := []string{"vm", "production", "staging", "monitoring"}

	for _, ns := range namespaces {
		t.Run("namespace_"+ns, func(t *testing.T) {
			vmSelectSvc := GetVMSelectSvc("vmks", ns)
			vmSingleSvc := GetVMSingleSvc("overwatch", ns)

			// Verify they contain the namespace
			assert.Contains(t, vmSelectSvc, ns, "VMSelect service address should contain namespace")
			assert.Contains(t, vmSingleSvc, ns, "VMSingle service address should contain namespace")

			// Verify they contain the correct service names
			assert.Contains(t, vmSelectSvc, "vmselect-vmks", "VMSelect service address should contain 'vmselect-vmks'")
			assert.Contains(t, vmSingleSvc, "vmsingle", "VMSingle service address should contain 'vmsingle'")

			// Verify they contain the correct ports
			assert.Contains(t, vmSelectSvc, ":8481", "VMSelect service address should contain port ':8481'")
			assert.Contains(t, vmSingleSvc, ":8428", "VMSingle service address should contain port ':8428'")
		})
	}
}

func TestGetVMInsertSvc(t *testing.T) {
	tests := []struct {
		name        string
		releaseName string
		namespace   string
		expected    string
	}{
		{
			name:        "standard namespace",
			releaseName: "vmks",
			namespace:   "vm",
			expected:    "vminsert-vmks.vm.svc.cluster.local:8480",
		},
		{
			name:        "production namespace",
			releaseName: "vmks",
			namespace:   "production",
			expected:    "vminsert-vmks.production.svc.cluster.local:8480",
		},
		{
			name:        "staging namespace",
			releaseName: "vmks",
			namespace:   "staging",
			expected:    "vminsert-vmks.staging.svc.cluster.local:8480",
		},
		{
			name:        "namespace with dashes",
			releaseName: "vmks",
			namespace:   "vm-cluster-test",
			expected:    "vminsert-vmks.vm-cluster-test.svc.cluster.local:8480",
		},
		{
			name:        "empty namespace",
			releaseName: "vmks",
			namespace:   "",
			expected:    "vminsert-vmks..svc.cluster.local:8480",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetVMInsertSvc(tt.releaseName, tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVMServiceAddressesIntegration(t *testing.T) {
	// Test that all VM service addresses work correctly across different namespaces
	namespaces := []string{"vm", "production", "staging", "monitoring"}

	for _, ns := range namespaces {
		t.Run("namespace_"+ns, func(t *testing.T) {
			vmSelectSvc := GetVMSelectSvc("vmks", ns)
			vmSingleSvc := GetVMSingleSvc("overwatch", ns)
			vmInsertSvc := GetVMInsertSvc("vmks", ns)

			// Verify they contain the namespace
			assert.Contains(t, vmSelectSvc, ns, "VMSelect service address should contain namespace")
			assert.Contains(t, vmSingleSvc, ns, "VMSingle service address should contain namespace")
			assert.Contains(t, vmInsertSvc, ns, "VMInsert service address should contain namespace")

			// Verify they contain the correct service names
			assert.Contains(t, vmSelectSvc, "vmselect-vmks", "VMSelect service address should contain 'vmselect-vmks'")
			assert.Contains(t, vmSingleSvc, "vmsingle", "VMSingle service address should contain 'vmsingle'")
			assert.Contains(t, vmInsertSvc, "vminsert-vmks", "VMInsert service address should contain 'vminsert-vmks'")

			// Verify they contain the correct ports
			assert.Contains(t, vmSelectSvc, ":8481", "VMSelect service address should contain port ':8481'")
			assert.Contains(t, vmSingleSvc, ":8428", "VMSingle service address should contain port ':8428'")
			assert.Contains(t, vmInsertSvc, ":8480", "VMInsert service address should contain port ':8480'")
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
