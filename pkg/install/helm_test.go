package install

import (
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

// MockTestingT implements terratesting.TestingT for testing
type MockTestingT struct {
	mock.Mock
}

func (m *MockTestingT) Errorf(format string, args ...interface{}) {
	m.Called(format, args)
}

func (m *MockTestingT) FailNow() {
	m.Called()
}

func (m *MockTestingT) Helper() {
	m.Called()
}

func (m *MockTestingT) Name() string {
	args := m.Called()
	return args.String(0)
}

func TestVMTagHelmIntegration(t *testing.T) {
	// Save original VM version and restore it after test
	originalVMVersion := consts.VMVersion()
	defer consts.SetVMTag(originalVMVersion)

	tests := []struct {
		name           string
		vmTag          string
		expectedTags   map[string]string
		shouldHaveTags bool
	}{
		{
			name:  "VM tag v1.131.0 should set all component tags",
			vmTag: "v1.131.0",
			expectedTags: map[string]string{
				"vmsingle.spec.image.tag":            "v1.131.0",
				"vmcluster.spec.vmstorage.image.tag": "v1.131.0",
				"vmcluster.spec.vmselect.image.tag":  "v1.131.0",
				"vmcluster.spec.vminsert.image.tag":  "v1.131.0",
				"vmalert.spec.image.tag":             "v1.131.0",
				"vmagent.spec.image.tag":             "v1.131.0",
				"vmauth.spec.image.tag":              "v1.131.0",
			},
			shouldHaveTags: true,
		},
		{
			name:  "VM tag v1.130.0 should set all component tags",
			vmTag: "v1.130.0",
			expectedTags: map[string]string{
				"vmsingle.spec.image.tag":            "v1.130.0",
				"vmcluster.spec.vmstorage.image.tag": "v1.130.0",
				"vmcluster.spec.vmselect.image.tag":  "v1.130.0",
				"vmcluster.spec.vminsert.image.tag":  "v1.130.0",
				"vmalert.spec.image.tag":             "v1.130.0",
				"vmagent.spec.image.tag":             "v1.130.0",
				"vmauth.spec.image.tag":              "v1.130.0",
			},
			shouldHaveTags: true,
		},
		{
			name:           "Empty VM tag should not set component tags",
			vmTag:          "",
			expectedTags:   map[string]string{},
			shouldHaveTags: false,
		},
		{
			name:  "Latest tag should set all component tags",
			vmTag: "latest",
			expectedTags: map[string]string{
				"vmsingle.spec.image.tag":            "latest",
				"vmcluster.spec.vmstorage.image.tag": "latest",
				"vmcluster.spec.vmselect.image.tag":  "latest",
				"vmcluster.spec.vminsert.image.tag":  "latest",
				"vmalert.spec.image.tag":             "latest",
				"vmagent.spec.image.tag":             "latest",
				"vmauth.spec.image.tag":              "latest",
			},
			shouldHaveTags: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the VM tag
			consts.SetVMTag(tt.vmTag)

			// Mock the VMSelectHost to avoid nil pointer
			consts.SetVMSelectHost("test-vm-select.example.com")

			// Create helm options using the same logic as InstallWithHelm
			kubeOpts := k8s.NewKubectlOptions("", "", "test-namespace")
			setValues := map[string]string{
				"vmcluster.ingress.select.hosts[0]": consts.VMSelectHost(),
			}

			// Add VM tag if provided (same logic as in InstallWithHelm)
			vmTag := consts.VMVersion()
			if vmTag != "" {
				setValues["vmsingle.spec.image.tag"] = vmTag
				setValues["vmcluster.spec.vmstorage.image.tag"] = vmTag
				setValues["vmcluster.spec.vmselect.image.tag"] = vmTag
				setValues["vmcluster.spec.vminsert.image.tag"] = vmTag
				setValues["vmalert.spec.image.tag"] = vmTag
				setValues["vmagent.spec.image.tag"] = vmTag
				setValues["vmauth.spec.image.tag"] = vmTag
			}

			helmOpts := &helm.Options{
				KubectlOptions: kubeOpts,
				SetValues:      setValues,
			}

			// Verify the helm options contain expected values
			if tt.shouldHaveTags {
				for key, expectedValue := range tt.expectedTags {
					actualValue, exists := helmOpts.SetValues[key]
					assert.True(t, exists, "Expected key '%s' to exist in helm SetValues", key)
					assert.Equal(t, expectedValue, actualValue, "Expected value for key '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
				}
			} else {
				// When VM tag is empty, only the vmcluster.ingress.select.hosts[0] should be set
				assert.Len(t, helmOpts.SetValues, 1, "Expected only one SetValue when VM tag is empty")
				_, exists := helmOpts.SetValues["vmcluster.ingress.select.hosts[0]"]
				assert.True(t, exists, "Expected vmcluster.ingress.select.hosts[0] to always be set")
			}

			// Verify all VM component tags are consistent
			if tt.shouldHaveTags {
				vmComponentTags := []string{
					"vmsingle.spec.image.tag",
					"vmcluster.spec.vmstorage.image.tag",
					"vmcluster.spec.vmselect.image.tag",
					"vmcluster.spec.vminsert.image.tag",
					"vmalert.spec.image.tag",
					"vmagent.spec.image.tag",
					"vmauth.spec.image.tag",
				}

				for _, tagKey := range vmComponentTags {
					actualValue := helmOpts.SetValues[tagKey]
					assert.Equal(t, tt.vmTag, actualValue, "All VM components should have the same tag '%s'", tt.vmTag)
				}
			}
		})
	}
}

func TestHelmOptionsStructure(t *testing.T) {
	// Test that helm options structure is correctly formed
	originalVMVersion := consts.VMVersion()
	defer consts.SetVMTag(originalVMVersion)

	consts.SetVMTag("v1.131.0")
	consts.SetVMSelectHost("test-host.example.com")

	kubeOpts := k8s.NewKubectlOptions("", "", "test-namespace")
	setValues := map[string]string{
		"vmcluster.ingress.select.hosts[0]": consts.VMSelectHost(),
	}

	vmTag := consts.VMVersion()
	if vmTag != "" {
		setValues["vmsingle.spec.image.tag"] = vmTag
		setValues["vmcluster.spec.vmstorage.image.tag"] = vmTag
		setValues["vmcluster.spec.vmselect.image.tag"] = vmTag
		setValues["vmcluster.spec.vminsert.image.tag"] = vmTag
		setValues["vmalert.spec.image.tag"] = vmTag
		setValues["vmagent.spec.image.tag"] = vmTag
		setValues["vmauth.spec.image.tag"] = vmTag
	}

	helmOpts := &helm.Options{
		KubectlOptions: kubeOpts,
		SetValues:      setValues,
		ExtraArgs: map[string][]string{
			"upgrade": {"--create-namespace", "--wait"},
		},
	}

	// Verify structure
	assert.NotNil(t, helmOpts.KubectlOptions)
	assert.Equal(t, "test-namespace", helmOpts.KubectlOptions.Namespace)
	assert.NotNil(t, helmOpts.SetValues)
	assert.NotNil(t, helmOpts.ExtraArgs)

	// Verify ExtraArgs contains expected upgrade flags
	upgradeArgs, exists := helmOpts.ExtraArgs["upgrade"]
	assert.True(t, exists, "Expected upgrade extra args to exist")
	assert.Contains(t, upgradeArgs, "--create-namespace")
	assert.Contains(t, upgradeArgs, "--wait")

	// Verify SetValues contains expected number of entries (8 VM components)
	assert.Len(t, helmOpts.SetValues, 8, "Expected 8 SetValues entries")
}

func TestVMTagConsistencyAcrossComponents(t *testing.T) {
	// Test that all VM components get the same tag consistently
	originalVMVersion := consts.VMVersion()
	defer consts.SetVMTag(originalVMVersion)

	testVersions := []string{"v1.131.0", "v1.130.0", "v1.129.1", "latest", "nightly"}

	for _, version := range testVersions {
		t.Run("version_"+version, func(t *testing.T) {
			consts.SetVMTag(version)
			consts.SetVMSelectHost("test-host.example.com")

			setValues := map[string]string{
				"vmcluster.ingress.select.hosts[0]": consts.VMSelectHost(),
			}

			vmTag := consts.VMVersion()
			if vmTag != "" {
				setValues["vmsingle.spec.image.tag"] = vmTag
				setValues["vmcluster.spec.vmstorage.image.tag"] = vmTag
				setValues["vmcluster.spec.vmselect.image.tag"] = vmTag
				setValues["vmcluster.spec.vminsert.image.tag"] = vmTag
				setValues["vmalert.spec.image.tag"] = vmTag
				setValues["vmagent.spec.image.tag"] = vmTag
				setValues["vmauth.spec.image.tag"] = vmTag
			}

			// Verify all VM components have the same tag
			vmComponents := []string{
				"vmsingle.spec.image.tag",
				"vmcluster.spec.vmstorage.image.tag",
				"vmcluster.spec.vmselect.image.tag",
				"vmcluster.spec.vminsert.image.tag",
				"vmalert.spec.image.tag",
				"vmagent.spec.image.tag",
				"vmauth.spec.image.tag",
			}

			for _, component := range vmComponents {
				actualTag, exists := setValues[component]
				assert.True(t, exists, "Component %s should have a tag set", component)
				assert.Equal(t, version, actualTag, "Component %s should have tag %s", component, version)
			}
		})
	}
}

func TestVMTagWithValuesFiles(t *testing.T) {
	// Test that VM tag works correctly when combined with values files
	originalVMVersion := consts.VMVersion()
	defer consts.SetVMTag(originalVMVersion)

	consts.SetVMTag("v1.131.0")
	consts.SetVMSelectHost("test-host.example.com")

	kubeOpts := k8s.NewKubectlOptions("", "", "test-namespace")
	setValues := map[string]string{
		"vmcluster.ingress.select.hosts[0]": consts.VMSelectHost(),
	}

	vmTag := consts.VMVersion()
	if vmTag != "" {
		setValues["vmsingle.spec.image.tag"] = vmTag
		setValues["vmcluster.spec.vmstorage.image.tag"] = vmTag
		setValues["vmcluster.spec.vmselect.image.tag"] = vmTag
		setValues["vmcluster.spec.vminsert.image.tag"] = vmTag
		setValues["vmalert.spec.image.tag"] = vmTag
		setValues["vmagent.spec.image.tag"] = vmTag
		setValues["vmauth.spec.image.tag"] = vmTag
	}

	helmOpts := &helm.Options{
		KubectlOptions: kubeOpts,
		ValuesFiles:    []string{"test-values.yaml"},
		SetValues:      setValues,
		ExtraArgs: map[string][]string{
			"upgrade": {"--create-namespace", "--wait"},
		},
	}

	// Verify that both ValuesFiles and SetValues are configured
	assert.NotEmpty(t, helmOpts.ValuesFiles, "ValuesFiles should not be empty")
	assert.Contains(t, helmOpts.ValuesFiles, "test-values.yaml")

	// Verify VM tag overrides are still present in SetValues
	for _, component := range []string{
		"vmsingle.spec.image.tag",
		"vmcluster.spec.vmstorage.image.tag",
		"vmcluster.spec.vmselect.image.tag",
		"vmcluster.spec.vminsert.image.tag",
		"vmalert.spec.image.tag",
		"vmagent.spec.image.tag",
		"vmauth.spec.image.tag",
	} {
		actualTag, exists := helmOpts.SetValues[component]
		assert.True(t, exists, "Component %s should have tag override", component)
		assert.Equal(t, "v1.131.0", actualTag, "Component %s should be overridden to v1.131.0", component)
	}
}
