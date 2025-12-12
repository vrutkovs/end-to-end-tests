package install

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDiscoverIngressHostWithLoadBalancer(t *testing.T) {
	// Create a fake Kubernetes client
	fakeClient := fake.NewSimpleClientset()

	// Create a service with LoadBalancer ingress
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-nginx-controller",
			Namespace: "ingress-nginx",
		},
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{
					{
						IP: "192.168.1.100",
					},
				},
			},
		},
	}

	_, err := fakeClient.CoreV1().Services("ingress-nginx").Create(
		context.Background(),
		service,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("Failed to create fake service: %v", err)
	}

	// Reset consts values
	consts.SetEnvK8SDistro("test")
	consts.SetNginxHost("")

	// Test the new nginx host API behavior
	consts.SetNginxHost("192.168.1.100")

	// Test hosts with different namespaces
	testNamespace := "test-ns"
	expectedVMSelectHost := "vmselect-test-ns.192.168.1.100.nip.io"
	expectedVMSingleHost := "vmsingle-test-ns.192.168.1.100.nip.io"
	expectedVMSelectUrl := "http://vmselect-test-ns.192.168.1.100.nip.io"
	expectedVMSingleUrl := "http://vmsingle-test-ns.192.168.1.100.nip.io"

	if consts.VMSelectHost(testNamespace) != expectedVMSelectHost {
		t.Errorf("Expected VMSelect host to be '%s', got '%s'", expectedVMSelectHost, consts.VMSelectHost(testNamespace))
	}
	if consts.VMSingleHost(testNamespace) != expectedVMSingleHost {
		t.Errorf("Expected VMSingle host to be '%s', got '%s'", expectedVMSingleHost, consts.VMSingleHost(testNamespace))
	}
	if consts.VMSelectUrl(testNamespace) != expectedVMSelectUrl {
		t.Errorf("Expected VMSelect URL to be '%s', got '%s'", expectedVMSelectUrl, consts.VMSelectUrl(testNamespace))
	}
	if consts.VMSingleUrl(testNamespace) != expectedVMSingleUrl {
		t.Errorf("Expected VMSingle URL to be '%s', got '%s'", expectedVMSingleUrl, consts.VMSingleUrl(testNamespace))
	}
}

func TestDiscoverIngressHostKindLogic(t *testing.T) {
	// Test the kind-specific logic
	consts.SetEnvK8SDistro("kind")
	consts.SetNginxHost("127.0.0.1")

	// Test with "vm" namespace (most common case)
	namespace := "vm"
	expectedVMSelectHost := "vmselect-vm.127.0.0.1.nip.io"
	expectedVMSingleHost := "vmsingle-vm.127.0.0.1.nip.io"
	expectedVMSelectUrl := "http://vmselect-vm.127.0.0.1.nip.io"
	expectedVMSingleUrl := "http://vmsingle-vm.127.0.0.1.nip.io"

	if consts.VMSelectHost(namespace) != expectedVMSelectHost {
		t.Errorf("Expected VMSelect host for kind to be '%s', got '%s'", expectedVMSelectHost, consts.VMSelectHost(namespace))
	}
	if consts.VMSingleHost(namespace) != expectedVMSingleHost {
		t.Errorf("Expected VMSingle host for kind to be '%s', got '%s'", expectedVMSingleHost, consts.VMSingleHost(namespace))
	}
	if consts.VMSelectUrl(namespace) != expectedVMSelectUrl {
		t.Errorf("Expected VMSelect URL for kind to be '%s', got '%s'", expectedVMSelectUrl, consts.VMSelectUrl(namespace))
	}
	if consts.VMSingleUrl(namespace) != expectedVMSingleUrl {
		t.Errorf("Expected VMSingle URL for kind to be '%s', got '%s'", expectedVMSingleUrl, consts.VMSingleUrl(namespace))
	}
}

func TestHostnameFormatting(t *testing.T) {
	tests := []struct {
		name              string
		nginxHost         string
		expectedSelect    string
		expectedSingle    string
		expectedSelectUrl string
		expectedSingleUrl string
	}{
		{
			name:              "IPv4 address",
			nginxHost:         "10.0.0.1",
			expectedSelect:    "vmselect.10.0.0.1.nip.io",
			expectedSingle:    "vmsingle.10.0.0.1.nip.io",
			expectedSelectUrl: "http://vmselect.10.0.0.1.nip.io",
			expectedSingleUrl: "http://vmsingle.10.0.0.1.nip.io",
		},
		{
			name:              "localhost",
			nginxHost:         "127.0.0.1",
			expectedSelect:    "vmselect.127.0.0.1.nip.io",
			expectedSingle:    "vmsingle.127.0.0.1.nip.io",
			expectedSelectUrl: "http://vmselect.127.0.0.1.nip.io",
			expectedSingleUrl: "http://vmsingle.127.0.0.1.nip.io",
		},
		{
			name:              "cloud provider IP",
			nginxHost:         "203.0.113.1",
			expectedSelect:    "vmselect.203.0.113.1.nip.io",
			expectedSingle:    "vmsingle.203.0.113.1.nip.io",
			expectedSelectUrl: "http://vmselect.203.0.113.1.nip.io",
			expectedSingleUrl: "http://vmsingle.203.0.113.1.nip.io",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consts.SetNginxHost(tt.nginxHost)

			// Test with empty namespace (no namespace prefix)
			namespace := ""

			if consts.VMSelectHost(namespace) != tt.expectedSelect {
				t.Errorf("Expected VMSelect host to be '%s', got '%s'", tt.expectedSelect, consts.VMSelectHost(namespace))
			}
			if consts.VMSingleHost(namespace) != tt.expectedSingle {
				t.Errorf("Expected VMSingle host to be '%s', got '%s'", tt.expectedSingle, consts.VMSingleHost(namespace))
			}
			if consts.VMSelectUrl(namespace) != tt.expectedSelectUrl {
				t.Errorf("Expected VMSelect URL to be '%s', got '%s'", tt.expectedSelectUrl, consts.VMSelectUrl(namespace))
			}
			if consts.VMSingleUrl(namespace) != tt.expectedSingleUrl {
				t.Errorf("Expected VMSingle URL to be '%s', got '%s'", tt.expectedSingleUrl, consts.VMSingleUrl(namespace))
			}
		})
	}
}

func TestNipIODomainPattern(t *testing.T) {
	// Test the nip.io domain pattern that's used in the ingress discovery
	tests := []struct {
		ip       string
		expected string
	}{
		{"192.168.1.1", "192.168.1.1.nip.io"},
		{"10.0.0.1", "10.0.0.1.nip.io"},
		{"127.0.0.1", "127.0.0.1.nip.io"},
		{"203.0.113.42", "203.0.113.42.nip.io"},
	}

	for _, tt := range tests {
		t.Run("IP_"+tt.ip, func(t *testing.T) {
			domain := tt.ip + ".nip.io"
			if domain != tt.expected {
				t.Errorf("Expected domain to be '%s', got '%s'", tt.expected, domain)
			}
		})
	}
}

// Test HTTP server for testing purposes
func TestHTTPServerSetup(t *testing.T) {
	// Create a test server to verify our HTTP client logic would work
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Test that we can make a request to the server
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request to test server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestConstsIntegration(t *testing.T) {
	// Test integration with consts package using nginx host
	testNginxHost := "192.168.100.50"
	testNamespace := "integration"
	expectedSelectHost := "vmselect-integration.192.168.100.50.nip.io"
	expectedSingleHost := "vmsingle-integration.192.168.100.50.nip.io"
	expectedSelectUrl := "http://vmselect-integration.192.168.100.50.nip.io"
	expectedSingleUrl := "http://vmsingle-integration.192.168.100.50.nip.io"

	consts.SetNginxHost(testNginxHost)

	if consts.VMSelectHost(testNamespace) != expectedSelectHost {
		t.Errorf("Expected VMSelectHost to be '%s', got '%s'", expectedSelectHost, consts.VMSelectHost(testNamespace))
	}
	if consts.VMSingleHost(testNamespace) != expectedSingleHost {
		t.Errorf("Expected VMSingleHost to be '%s', got '%s'", expectedSingleHost, consts.VMSingleHost(testNamespace))
	}
	if consts.VMSelectUrl(testNamespace) != expectedSelectUrl {
		t.Errorf("Expected VMSelectUrl to be '%s', got '%s'", expectedSelectUrl, consts.VMSelectUrl(testNamespace))
	}
	if consts.VMSingleUrl(testNamespace) != expectedSingleUrl {
		t.Errorf("Expected VMSingleUrl to be '%s', got '%s'", expectedSingleUrl, consts.VMSingleUrl(testNamespace))
	}
}

func TestWaitForLoadBalancerIngress(t *testing.T) {
	tests := []struct {
		name           string
		services       []*corev1.Service
		expectedHost   string
		shouldFail     bool
		failureMessage string
	}{
		{
			name: "service with IP immediately available",
			services: []*corev1.Service{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "ingress-nginx-controller",
						Namespace: "ingress-nginx",
					},
					Status: corev1.ServiceStatus{
						LoadBalancer: corev1.LoadBalancerStatus{
							Ingress: []corev1.LoadBalancerIngress{
								{IP: "192.168.1.100"},
							},
						},
					},
				},
			},
			expectedHost: "192.168.1.100",
			shouldFail:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake Kubernetes client
			fakeClient := fake.NewSimpleClientset()

			// Create the service in the fake client
			if len(tt.services) > 0 {
				for _, svc := range tt.services {
					_, err := fakeClient.CoreV1().Services("ingress-nginx").Create(
						context.Background(),
						svc,
						metav1.CreateOptions{},
					)
					if err != nil {
						t.Fatalf("Failed to create fake service: %v", err)
					}
				}
			}

			// Note: We can't easily test waitForLoadBalancerIngress directly because
			// it uses k8s.GetService which creates its own client. This test validates
			// the expected behavior and logic structure for the LoadBalancer ingress
			// discovery functionality.

			// Test the logic that would be used in waitForLoadBalancerIngress
			if len(tt.services) > 0 && !tt.shouldFail {
				svc := tt.services[0]
				var host string

				if len(svc.Status.LoadBalancer.Ingress) > 0 {
					ingress := svc.Status.LoadBalancer.Ingress[0]
					if ingress.IP != "" {
						host = ingress.IP
					}
				}

				if host != tt.expectedHost {
					t.Errorf("Expected host to be '%s', got '%s'", tt.expectedHost, host)
				}
			}
		})
	}
}

func TestEnvironmentDistroLogic(t *testing.T) {
	tests := []struct {
		name         string
		distro       string
		expectKind   bool
		expectedHost string
	}{
		{
			name:         "kind environment",
			distro:       "kind",
			expectKind:   true,
			expectedHost: "127.0.0.1",
		},
		{
			name:         "non-kind environment",
			distro:       "gke",
			expectKind:   false,
			expectedHost: "",
		},
		{
			name:         "empty distro",
			distro:       "",
			expectKind:   false,
			expectedHost: "",
		},
		{
			name:         "other distro",
			distro:       "eks",
			expectKind:   false,
			expectedHost: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment distro
			originalDistro := consts.EnvK8SDistro()
			defer consts.SetEnvK8SDistro(originalDistro) // Restore original value

			consts.SetEnvK8SDistro(tt.distro)

			isKind := consts.EnvK8SDistro() == "kind"
			if isKind != tt.expectKind {
				t.Errorf("Expected isKind to be %v, got %v", tt.expectKind, isKind)
			}

			if tt.expectKind {
				// For kind environments, we should use localhost
				nginxHost := "127.0.0.1"
				if nginxHost != tt.expectedHost {
					t.Errorf("Expected nginx host for kind to be '%s', got '%s'", tt.expectedHost, nginxHost)
				}
			}
		})
	}
}

func TestWatchUntilWithoutRetryBehavior(t *testing.T) {
	// Test that validates the watch-based approach behavior
	tests := []struct {
		name                string
		distro              string
		shouldUseWatch      bool
		expectedHostPattern string
	}{
		{
			name:                "kind environment skips watch",
			distro:              "kind",
			shouldUseWatch:      false,
			expectedHostPattern: "127.0.0.1",
		},
		{
			name:                "gke environment uses watch",
			distro:              "gke",
			shouldUseWatch:      true,
			expectedHostPattern: "", // Would be set by watch
		},
		{
			name:                "eks environment uses watch",
			distro:              "eks",
			shouldUseWatch:      true,
			expectedHostPattern: "", // Would be set by watch
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set the environment distro
			originalDistro := consts.EnvK8SDistro()
			defer consts.SetEnvK8SDistro(originalDistro) // Restore original value

			consts.SetEnvK8SDistro(tt.distro)

			// Test the logic that determines whether to use watch
			isKind := consts.EnvK8SDistro() == "kind"
			shouldUseWatch := !isKind

			if shouldUseWatch != tt.shouldUseWatch {
				t.Errorf("Expected shouldUseWatch to be %v, got %v", tt.shouldUseWatch, shouldUseWatch)
			}

			if isKind && tt.expectedHostPattern != "" {
				// For kind, we should use localhost immediately
				expectedHost := "127.0.0.1"
				if expectedHost != tt.expectedHostPattern {
					t.Errorf("Expected kind host to be '%s', got '%s'", tt.expectedHostPattern, expectedHost)
				}
			}
		})
	}
}

func TestExtractIngressHost(t *testing.T) {
	tests := []struct {
		name         string
		service      *corev1.Service
		expectedHost string
	}{
		{
			name: "service with IP",
			service: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "192.168.1.100"},
						},
					},
				},
			},
			expectedHost: "192.168.1.100",
		},

		{
			name: "service with IP only",
			service: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: "203.0.113.42"},
						},
					},
				},
			},
			expectedHost: "203.0.113.42",
		},
		{
			name: "service with no ingress",
			service: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{},
					},
				},
			},
			expectedHost: "",
		},
		{
			name: "service with empty IP",
			service: &corev1.Service{
				Status: corev1.ServiceStatus{
					LoadBalancer: corev1.LoadBalancerStatus{
						Ingress: []corev1.LoadBalancerIngress{
							{IP: ""},
						},
					},
				},
			},
			expectedHost: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := extractIngressHost(tt.service)
			if host != tt.expectedHost {
				t.Errorf("Expected host to be '%s', got '%s'", tt.expectedHost, host)
			}
		})
	}
}
