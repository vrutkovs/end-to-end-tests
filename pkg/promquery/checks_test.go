package promquery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	terratesting "github.com/gruntwork-io/terratest/modules/testing"
)

type mockTestingT struct {
	failed   bool
	logs     []string
	errors   []string
	fatals   []string
	skipLogs []string
}

func (m *mockTestingT) Name() string {
	return "mock-test"
}

func (m *mockTestingT) Error(args ...interface{}) {
	m.failed = true
	m.errors = append(m.errors, fmt.Sprint(args...))
}

func (m *mockTestingT) Errorf(format string, args ...interface{}) {
	m.failed = true
	m.errors = append(m.errors, fmt.Sprintf(format, args...))
}

func (m *mockTestingT) Fail() {
	m.failed = true
}

func (m *mockTestingT) FailNow() {
	m.failed = true
	m.fatals = append(m.fatals, "FailNow called")
}

func (m *mockTestingT) Failed() bool {
	return m.failed
}

func (m *mockTestingT) Fatal(args ...interface{}) {
	m.failed = true
	m.fatals = append(m.fatals, fmt.Sprint(args...))
}

func (m *mockTestingT) Fatalf(format string, args ...interface{}) {
	m.failed = true
	m.fatals = append(m.fatals, fmt.Sprintf(format, args...))
}

func (m *mockTestingT) Log(args ...interface{}) {
	m.logs = append(m.logs, fmt.Sprint(args...))
}

func (m *mockTestingT) Logf(format string, args ...interface{}) {
	m.logs = append(m.logs, fmt.Sprintf(format, args...))
}

func (m *mockTestingT) Skip(args ...interface{}) {
	m.skipLogs = append(m.skipLogs, fmt.Sprint(args...))
}

func (m *mockTestingT) SkipNow() {
	m.skipLogs = append(m.skipLogs, "SkipNow called")
}

func (m *mockTestingT) Skipf(format string, args ...interface{}) {
	m.skipLogs = append(m.skipLogs, fmt.Sprintf(format, args...))
}

func (m *mockTestingT) Skipped() bool {
	return len(m.skipLogs) > 0
}

var _ terratesting.TestingT = (*mockTestingT)(nil)

func TestCheckNoAlertsFiring_NoAlerts(t *testing.T) {
	// Create a mock server that returns at least one alert with value 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/query" {
			t.Errorf("Expected path '/api/v1/query', got '%s'", r.URL.Path)
		}

		// Parse form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
			return
		}

		// Verify the query contains the expected exclusions
		query := r.Form.Get("query")
		expectedSubstring := `sum by (alertname) (vmalert_alerts_firing{alertname!~"InfoInhibitor|Watchdog"})`
		if query != expectedSubstring {
			t.Errorf("Expected query to be '%s', got '%s'", expectedSubstring, query)
		}

		// Return vector with non-firing alert (value = 0)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "TestAlert"},
						"value": [1234567890, "0"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should not fail when alert value is 0
	if mockTest.failed {
		t.Errorf("Test should not have failed, but got errors: %v, fatals: %v", mockTest.errors, mockTest.fatals)
	}
}

func TestCheckNoAlertsFiring_WithCustomExceptions(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Parse form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
			return
		}

		// Verify the query includes custom exceptions
		query := r.Form.Get("query")
		expectedSubstring := `sum by (alertname) (vmalert_alerts_firing{alertname!~"InfoInhibitor|Watchdog|CustomAlert|TestAlert"})`
		if query != expectedSubstring {
			t.Errorf("Expected query to be '%s', got '%s'", expectedSubstring, query)
		}

		// Return vector with non-firing alert (value = 0)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "SomeOtherAlert"},
						"value": [1234567890, "0"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()
	exceptions := []string{"CustomAlert", "TestAlert"}

	client.CheckNoAlertsFiring(ctx, mockTest, exceptions)

	if mockTest.failed {
		t.Errorf("Test should not have failed, but got errors: %v, fatals: %v", mockTest.errors, mockTest.fatals)
	}
}

func TestCheckNoAlertsFiring_WithFiringAlerts(t *testing.T) {
	// Create a mock server that returns firing alerts
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return vector with firing alerts
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "CriticalAlert"},
						"value": [1234567890, "1"]
					},
					{
						"metric": {"alertname": "WarningAlert"},
						"value": [1234567890, "2"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should fail because alerts are firing
	if !mockTest.failed {
		t.Error("Test should have failed when alerts are firing")
	}

	// Should have error messages about the firing alerts
	if len(mockTest.errors) == 0 {
		t.Error("Expected error messages about firing alerts")
	}
}

func TestCheckNoAlertsFiring_WithZeroValueAlerts(t *testing.T) {
	// Create a mock server that returns alerts with value 0 (not firing)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return vector with non-firing alerts (value = 0)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "ResolvedAlert"},
						"value": [1234567890, "0"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should not fail when alert value is 0
	if mockTest.failed {
		t.Errorf("Test should not have failed for zero-value alerts, but got errors: %v, fatals: %v", mockTest.errors, mockTest.fatals)
	}
}

func TestCheckNoAlertsFiring_QueryError(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should not fail when query errors (the method handles this case by not checking errors)
	if mockTest.failed {
		t.Errorf("Test should not have failed on query error, but got errors: %v, fatals: %v", mockTest.errors, mockTest.fatals)
	}
}

func TestCheckNoAlertsFiring_WrongResultType(t *testing.T) {
	// Create a mock server that returns wrong result type
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return matrix instead of vector
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": []
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	// This should cause a panic in the current implementation
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when result type is not vector")
		}
	}()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})
}

func TestCheckNoAlertsFiring_DefaultExceptions(t *testing.T) {
	tests := []struct {
		name              string
		customExceptions  []string
		expectedQueryPart string
	}{
		{
			name:              "only default exceptions",
			customExceptions:  []string{},
			expectedQueryPart: "InfoInhibitor|Watchdog",
		},
		{
			name:              "default and custom exceptions",
			customExceptions:  []string{"MyCustomAlert"},
			expectedQueryPart: "InfoInhibitor|Watchdog|MyCustomAlert",
		},
		{
			name:              "multiple custom exceptions",
			customExceptions:  []string{"Alert1", "Alert2", "Alert3"},
			expectedQueryPart: "InfoInhibitor|Watchdog|Alert1|Alert2|Alert3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Parse form data
				err := r.ParseForm()
				if err != nil {
					t.Errorf("Failed to parse form: %v", err)
					return
				}

				query := r.Form.Get("query")
				expectedQuery := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{alertname!~"%s"})`, tt.expectedQueryPart)

				if query != expectedQuery {
					t.Errorf("Expected query to be '%s', got '%s'", expectedQuery, query)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, `{
					"status": "success",
					"data": {
						"resultType": "vector",
						"result": [
							{
								"metric": {"alertname": "TestAlert"},
								"value": [1234567890, "0"]
							}
						]
					}
				}`)
			}))
			defer server.Close()

			client, err := NewPrometheusClient(server.URL)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			mockTest := &mockTestingT{}
			ctx := context.Background()

			client.CheckNoAlertsFiring(ctx, mockTest, tt.customExceptions)

			if mockTest.failed {
				t.Errorf("Test should not have failed, but got errors: %v", mockTest.errors)
			}
		})
	}
}

func TestCheckNoAlertsFiring_EmptyVectorShouldFail(t *testing.T) {
	// Create a mock server that returns empty vector - this should fail
	// because the function expects at least one result
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return empty vector
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should fail when vector is empty because function requires at least one result
	if !mockTest.failed {
		t.Error("Test should have failed for empty vector")
	}
}
