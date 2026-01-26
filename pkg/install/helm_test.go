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

func TestBuildVMK8StackValues(t *testing.T) {
	// Save original values
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	tests := []struct {
		name           string
		vmTag          string
		namespace      string
		nginxHost      string
		expectedTags   map[string]string
		shouldHaveTags bool
	}{
		{
			name:      "VM tag v1.131.0 with namespace should set all component tags with cluster suffix",
			vmTag:     "v1.131.0",
			namespace: "vm",
			nginxHost: "192.168.1.100",
			expectedTags: map[string]string{
				"vmcluster.ingress.select.hosts[0]":  "vmselect-vm.192.168.1.100.nip.io",
				"vmcluster.ingress.insert.hosts[0]":  "vminsert-vm.192.168.1.100.nip.io",
				"vmsingle.spec.image.tag":            "v1.131.0",
				"vmcluster.spec.vmstorage.image.tag": "v1.131.0-cluster",
				"vmcluster.spec.vmselect.image.tag":  "v1.131.0-cluster",
				"vmcluster.spec.vminsert.image.tag":  "v1.131.0-cluster",
				"vmalert.spec.image.tag":             "v1.131.0",
				"vmagent.spec.image.tag":             "v1.131.0",
				"vmauth.spec.image.tag":              "v1.131.0",
			},
			shouldHaveTags: true,
		},
		{
			name:      "VM tag v1.130.0 with empty namespace",
			vmTag:     "v1.130.0",
			namespace: "",
			nginxHost: "127.0.0.1",
			expectedTags: map[string]string{
				"vmcluster.ingress.select.hosts[0]":  "vmselect.127.0.0.1.nip.io",
				"vmcluster.ingress.insert.hosts[0]":  "vminsert.127.0.0.1.nip.io",
				"vmsingle.spec.image.tag":            "v1.130.0",
				"vmcluster.spec.vmstorage.image.tag": "v1.130.0-cluster",
				"vmcluster.spec.vmselect.image.tag":  "v1.130.0-cluster",
				"vmcluster.spec.vminsert.image.tag":  "v1.130.0-cluster",
				"vmalert.spec.image.tag":             "v1.130.0",
				"vmagent.spec.image.tag":             "v1.130.0",
				"vmauth.spec.image.tag":              "v1.130.0",
			},
			shouldHaveTags: true,
		},
		{
			name:      "Empty VM tag should only set ingress host",
			vmTag:     "",
			namespace: "test",
			nginxHost: "10.0.0.1",
			expectedTags: map[string]string{
				"vmcluster.ingress.select.hosts[0]": "vmselect-test.10.0.0.1.nip.io",
				"vmcluster.ingress.insert.hosts[0]": "vminsert-test.10.0.0.1.nip.io",
			},
			shouldHaveTags: false,
		},
		{
			name:      "Latest tag should set all component tags without cluster suffix",
			vmTag:     "latest",
			namespace: "production",
			nginxHost: "172.16.1.50",
			expectedTags: map[string]string{
				"vmcluster.ingress.select.hosts[0]":  "vmselect-production.172.16.1.50.nip.io",
				"vmcluster.ingress.insert.hosts[0]":  "vminsert-production.172.16.1.50.nip.io",
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
		{
			name:      "Nightly tag should set all component tags with cluster suffix",
			vmTag:     "nightly",
			namespace: "staging",
			nginxHost: "203.0.113.10",
			expectedTags: map[string]string{
				"vmcluster.ingress.select.hosts[0]":  "vmselect-staging.203.0.113.10.nip.io",
				"vmcluster.ingress.insert.hosts[0]":  "vminsert-staging.203.0.113.10.nip.io",
				"vmsingle.spec.image.tag":            "nightly",
				"vmcluster.spec.vmstorage.image.tag": "nightly-cluster",
				"vmcluster.spec.vmselect.image.tag":  "nightly-cluster",
				"vmcluster.spec.vminsert.image.tag":  "nightly-cluster",
				"vmalert.spec.image.tag":             "nightly",
				"vmagent.spec.image.tag":             "nightly",
				"vmauth.spec.image.tag":              "nightly",
			},
			shouldHaveTags: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consts.SetVMTag(tt.vmTag)
			consts.SetNginxHost(tt.nginxHost)

			setValues := buildVMK8StackValues(tt.namespace)

			for key, expectedValue := range tt.expectedTags {
				actualValue, exists := setValues[key]
				assert.True(t, exists, "Expected key '%s' to exist in setValues", key)
				assert.Equal(t, expectedValue, actualValue, "Expected value for key '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
			}

			if tt.shouldHaveTags {
				assert.Len(t, setValues, len(tt.expectedTags), "SetValues should contain exactly %d entries", len(tt.expectedTags))
			} else {
				assert.Len(t, setValues, 2, "SetValues should contain only ingress hosts when no VM tag is set")
			}
		})
	}
}

func TestBuildVMK8StackValuesConsistency(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	testVersions := []string{"v1.131.0", "v1.130.0", "v1.129.1", "nightly"}

	for _, version := range testVersions {
		t.Run("version_"+version, func(t *testing.T) {
			consts.SetVMTag(version)
			consts.SetNginxHost("192.168.1.200")

			setValues := buildVMK8StackValues("vm")

			expectedClusterTag := version + "-cluster"
			clusterComponents := []string{
				"vmcluster.spec.vmstorage.image.tag",
				"vmcluster.spec.vmselect.image.tag",
				"vmcluster.spec.vminsert.image.tag",
			}

			for _, component := range clusterComponents {
				actualTag, exists := setValues[component]
				assert.True(t, exists, "Cluster component %s should have a tag set", component)
				assert.Equal(t, expectedClusterTag, actualTag, "Cluster component %s should have tag %s", component, expectedClusterTag)
			}

			nonClusterComponents := []string{
				"vmsingle.spec.image.tag",
				"vmalert.spec.image.tag",
				"vmagent.spec.image.tag",
				"vmauth.spec.image.tag",
			}

			for _, component := range nonClusterComponents {
				actualTag, exists := setValues[component]
				assert.True(t, exists, "Non-cluster component %s should have a tag set", component)
				assert.Equal(t, version, actualTag, "Non-cluster component %s should have tag %s", component, version)
			}
		})
	}
}

func TestBuildVMK8StackValuesLatestSpecialCase(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	consts.SetVMTag("latest")
	consts.SetNginxHost("172.17.0.100")

	setValues := buildVMK8StackValues("vm")

	allComponents := []string{
		"vmsingle.spec.image.tag",
		"vmcluster.spec.vmstorage.image.tag",
		"vmcluster.spec.vmselect.image.tag",
		"vmcluster.spec.vminsert.image.tag",
		"vmalert.spec.image.tag",
		"vmagent.spec.image.tag",
		"vmauth.spec.image.tag",
	}

	for _, component := range allComponents {
		actualTag, exists := setValues[component]
		assert.True(t, exists, "Component %s should have a tag set", component)
		assert.Equal(t, "latest", actualTag, "Component %s should have tag 'latest'", component)
	}

	ingressHost, exists := setValues["vmcluster.ingress.select.hosts[0]"]
	assert.True(t, exists, "Ingress host should be set")
	assert.Equal(t, "vmselect-vm.172.17.0.100.nip.io", ingressHost)

	insertHost, exists := setValues["vmcluster.ingress.insert.hosts[0]"]
	assert.True(t, exists, "Insert ingress host should be set")
	assert.Equal(t, "vminsert-vm.172.17.0.100.nip.io", insertHost)
}

func TestBuildVMK8StackValuesWithEmptyNginxHost(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	consts.SetVMTag("v1.131.0")
	consts.SetNginxHost("")

	setValues := buildVMK8StackValues("vm")

	ingressHost, exists := setValues["vmcluster.ingress.select.hosts[0]"]
	assert.True(t, exists, "Ingress host key should exist")
	assert.Equal(t, "", ingressHost, "Ingress host should be empty string")

	insertHost, exists := setValues["vmcluster.ingress.insert.hosts[0]"]
	assert.True(t, exists, "Insert ingress host key should exist")
	assert.Equal(t, "", insertHost, "Insert ingress host should be empty string")

	assert.Equal(t, "v1.131.0", setValues["vmsingle.spec.image.tag"])
	assert.Equal(t, "v1.131.0-cluster", setValues["vmcluster.spec.vmstorage.image.tag"])
}

func TestBuildVMK8StackValuesReturnsCopy(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	consts.SetVMTag("v1.131.0")
	consts.SetNginxHost("198.51.100.42")

	setValues1 := buildVMK8StackValues("test")
	setValues2 := buildVMK8StackValues("test")

	assert.Equal(t, setValues1, setValues2, "Both calls should return maps with same content")

	setValues1["test.key"] = "modified"
	_, exists := setValues2["test.key"]
	assert.False(t, exists, "Modifying one map should not affect the other")
}

func TestInstallWithHelmUsesVMK8StackValues(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	consts.SetVMTag("v1.131.0")
	consts.SetNginxHost("10.10.10.10")

	setValues := buildVMK8StackValues("vm")

	kubeOpts := k8s.NewKubectlOptions("", "", "test-namespace")
	helmOpts := &helm.Options{
		KubectlOptions: kubeOpts,
		ValuesFiles:    []string{"test-values.yaml"},
		SetValues:      setValues,
		ExtraArgs: map[string][]string{
			"upgrade": {"--create-namespace", "--wait"},
		},
	}

	assert.NotNil(t, helmOpts.KubectlOptions)
	assert.Equal(t, "test-namespace", helmOpts.KubectlOptions.Namespace)
	assert.NotNil(t, helmOpts.SetValues)
	assert.NotNil(t, helmOpts.ExtraArgs)

	upgradeArgs, exists := helmOpts.ExtraArgs["upgrade"]
	assert.True(t, exists, "Expected upgrade extra args to exist")
	assert.Contains(t, upgradeArgs, "--create-namespace")
	assert.Contains(t, upgradeArgs, "--wait")

	expectedSetValues := map[string]string{
		"vmcluster.ingress.select.hosts[0]":  "vmselect-vm.10.10.10.10.nip.io",
		"vmcluster.ingress.insert.hosts[0]":  "vminsert-vm.10.10.10.10.nip.io",
		"vmsingle.spec.image.tag":            "v1.131.0",
		"vmcluster.spec.vmstorage.image.tag": "v1.131.0-cluster",
		"vmcluster.spec.vmselect.image.tag":  "v1.131.0-cluster",
		"vmcluster.spec.vminsert.image.tag":  "v1.131.0-cluster",
		"vmalert.spec.image.tag":             "v1.131.0",
		"vmagent.spec.image.tag":             "v1.131.0",
		"vmauth.spec.image.tag":              "v1.131.0",
	}

	assert.Len(t, helmOpts.SetValues, len(expectedSetValues), "SetValues should contain expected number of entries")

	for key, expectedValue := range expectedSetValues {
		actualValue, exists := helmOpts.SetValues[key]
		assert.True(t, exists, "Expected key '%s' to exist in helm SetValues", key)
		assert.Equal(t, expectedValue, actualValue, "Expected value for key '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
	}
}

func TestBuildVMDistributedValues(t *testing.T) {
	originalVMVersion := consts.VMVersion()
	originalNginxHost := consts.NginxHost()
	defer func() {
		consts.SetVMTag(originalVMVersion)
		consts.SetNginxHost(originalNginxHost)
	}()

	tests := []struct {
		name           string
		vmTag          string
		namespace      string
		nginxHost      string
		expectedTags   map[string]string
		shouldHaveTags bool
	}{
		{
			name:      "VM tag v1.131.0 with namespace should set all component tags with cluster suffix",
			vmTag:     "v1.131.0",
			namespace: "vm",
			nginxHost: "192.168.1.100",
			expectedTags: map[string]string{
				"read.global.vmauth.spec.ingress.host":      "vmselect-vm.192.168.1.100.nip.io",
				"write.global.vmauth.spec.ingress.host":     "vminsert-vm.192.168.1.100.nip.io",
				"common.vmcluster.spec.vmstorage.image.tag": "v1.131.0-cluster",
				"common.vmcluster.spec.vmselect.image.tag":  "v1.131.0-cluster",
				"common.vmcluster.spec.vminsert.image.tag":  "v1.131.0-cluster",
				"common.vmalert.spec.image.tag":             "v1.131.0",
				"common.vmagent.spec.image.tag":             "v1.131.0",
				"common.vmauth.spec.image.tag":              "v1.131.0",
			},
			shouldHaveTags: true,
		},
		{
			name:      "Empty VM tag should only set ingress hosts",
			vmTag:     "",
			namespace: "test",
			nginxHost: "10.0.0.1",
			expectedTags: map[string]string{
				"read.global.vmauth.spec.ingress.host":  "vmselect-test.10.0.0.1.nip.io",
				"write.global.vmauth.spec.ingress.host": "vminsert-test.10.0.0.1.nip.io",
			},
			shouldHaveTags: false,
		},
		{
			name:      "Latest tag should set all component tags without cluster suffix",
			vmTag:     "latest",
			namespace: "production",
			nginxHost: "172.16.1.50",
			expectedTags: map[string]string{
				"read.global.vmauth.spec.ingress.host":      "vmselect-production.172.16.1.50.nip.io",
				"write.global.vmauth.spec.ingress.host":     "vminsert-production.172.16.1.50.nip.io",
				"common.vmcluster.spec.vmstorage.image.tag": "latest",
				"common.vmcluster.spec.vmselect.image.tag":  "latest",
				"common.vmcluster.spec.vminsert.image.tag":  "latest",
				"common.vmalert.spec.image.tag":             "latest",
				"common.vmagent.spec.image.tag":             "latest",
				"common.vmauth.spec.image.tag":              "latest",
			},
			shouldHaveTags: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consts.SetVMTag(tt.vmTag)
			consts.SetNginxHost(tt.nginxHost)

			setValues := buildVMDistributedValues(tt.namespace)

			for key, expectedValue := range tt.expectedTags {
				actualValue, exists := setValues[key]
				assert.True(t, exists, "Expected key '%s' to exist in setValues", key)
				assert.Equal(t, expectedValue, actualValue, "Expected value for key '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
			}

			if tt.shouldHaveTags {
				assert.Len(t, setValues, len(tt.expectedTags), "SetValues should contain exactly %d entries", len(tt.expectedTags))
			} else {
				assert.Len(t, setValues, 2, "SetValues should contain only ingress hosts when no VM tag is set")
			}
		})
	}
}
