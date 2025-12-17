package promquery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
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

// Ensure mockTestingT implements a testing interface
var _ interface {
	Name() string
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Skip(args ...interface{})
	SkipNow()
	Skipf(format string, args ...interface{})
	Skipped() bool
} = (*mockTestingT)(nil)

func TestCheckNoAlertsFiring_NoAlerts(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns at least one alert with value 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		assert.Equal(t, "POST", r.Method, "Expected POST request")
		assert.Equal(t, "/api/v1/query", r.URL.Path, "Expected correct API path")

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err, "Failed to parse form data")

		// Verify the query contains the expected exclusions
		query := r.Form.Get("query")
		expectedSubstring := `sum by (alertname) (vmalert_alerts_firing{alertname!~"InfoInhibitor|Watchdog"})`
		assert.Equal(t, expectedSubstring, query, "Query should match expected format")

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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should not fail when alert value is 0
	assert.False(t, mockTest.failed, "Test should not have failed when alert value is 0")
	assert.Empty(t, mockTest.errors, "Should not have any errors")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors")
}

func TestCheckNoAlertsFiring_WithCustomExceptions(t *testing.T) {
	t.Parallel()
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
		expectedSubstring := `sum by (alertname) (vmalert_alerts_firing{alertname!~"InfoInhibitor|Watchdog|TestAlert1|TestAlert2"})`
		assert.Equal(t, expectedSubstring, query, "Query should include custom exceptions")

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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{"TestAlert1", "TestAlert2"})

	// Should not fail when alert value is 0 with custom exceptions
	assert.False(t, mockTest.failed, "Test should not have failed with custom exceptions")
	assert.Empty(t, mockTest.errors, "Should not have any errors")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors")
}

func TestCheckNoAlertsFiring_WithFiringAlerts(t *testing.T) {
	t.Parallel()
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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should fail when alert value is > 0
	assert.True(t, mockTest.failed, "Test should have failed when alerts are firing")
	assert.NotEmpty(t, mockTest.errors, "Should have error messages when alerts are firing")

	// Should have error messages about the firing alerts
	if len(mockTest.errors) == 0 {
		t.Error("Expected error messages about firing alerts")
	}
}

func TestCheckNoAlertsFiring_WithZeroValueAlerts(t *testing.T) {
	t.Parallel()
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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{"TestAlert1", "TestAlert2"})

	// Should not fail when alert value is 0 with custom exceptions
	assert.False(t, mockTest.failed, "Test should not have failed with custom exceptions")
	assert.Empty(t, mockTest.errors, "Should not have any errors")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors")
}

func TestCheckNoAlertsFiring_QueryError(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should not fail when alert value is 0
	assert.False(t, mockTest.failed, "Test should not have failed with zero value alerts")
	assert.Empty(t, mockTest.errors, "Should not have any errors with zero value alerts")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors with zero value alerts")
}

func TestCheckNoAlertsFiring_WrongResultType(t *testing.T) {
	t.Parallel()
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

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Expect the mock test to record a failure when result type is not vector
	if !mockTest.failed {
		t.Error("Expected test to have failed when result type is not vector")
	}
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
			t.Parallel()
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

func TestCheckAlertIsFiring_AlertFiring(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns a firing alert
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method and path
		assert.Equal(t, "POST", r.Method, "Expected POST request")
		assert.Equal(t, "/api/v1/query", r.URL.Path, "Expected correct API path")

		// Parse form data
		err := r.ParseForm()
		require.NoError(t, err, "Failed to parse form data")

		// Verify the query is for the specific alert
		query := r.Form.Get("query")
		expectedQuery := `vmalert_alerts_firing{alertname="TestAlert"}`
		if query != expectedQuery {
			t.Errorf("Expected query to be '%s', got '%s'", expectedQuery, query)
		}

		// Return vector with firing alert (value > 0)
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
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "TestAlert")

	// Should not fail when alert is firing
	assert.False(t, mockTest.failed, "Test should not have failed when alert is firing")
	assert.Empty(t, mockTest.errors, "Should not have any errors when alert is firing")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors when alert is firing")
}

func TestCheckAlertIsFiring_AlertNotFiring(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns a non-firing alert
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return vector with non-firing alert (value = 0)
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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "TestAlert")

	// Should fail when alert is not firing
	assert.True(t, mockTest.failed, "Test should have failed when alert is not firing")
	assert.NotEmpty(t, mockTest.errors, "Should have error when alert is not firing")

	// Should have error message about alert not firing
	if len(mockTest.errors) == 0 {
		t.Error("Expected error message about alert not firing")
	}
}

func TestCheckAlertIsFiring_AlertNotFound(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns empty results
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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "NonExistentAlert")

	// Should fail when alert is not found
	assert.True(t, mockTest.failed, "Test should have failed when alert is not found")
	assert.NotEmpty(t, mockTest.errors, "Should have error when alert is not found")

	// Should have error message about alert not being present
	if len(mockTest.errors) == 0 {
		t.Error("Expected error message about alert not being present")
	}
}

func TestCheckAlertIsFiring_MultipleAlerts(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns multiple alerts, including the target one
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return vector with multiple alerts
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "WarningAlert"},
						"value": [1234567890, "1"]
					},
					{
						"metric": {"alertname": "CriticalAlert"},
						"value": [1234567890, "2"]
					},
					{
						"metric": {"alertname": "InfoAlert"},
						"value": [1234567890, "0"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "TestAlert")

	// Should not fail - the function should handle multiple alerts gracefully
	assert.False(t, mockTest.failed, "Test should not have failed with multiple alerts")
	assert.Empty(t, mockTest.errors, "Should not have any errors with multiple alerts")
	assert.Empty(t, mockTest.fatals, "Should not have any fatal errors with multiple alerts")
}

func TestCheckAlertIsFiring_QueryError(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "TestAlert")

	// Should fail due to query error
	assert.True(t, mockTest.failed, "Test should have failed due to query error")
	assert.NotEmpty(t, mockTest.fatals, "Should have fatal error due to query failure")
}

func TestCheckAlertIsFiring_WrongResultType(t *testing.T) {
	t.Parallel()
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

	client.CheckAlertIsFiring(ctx, mockTest, "TestAlert")

	// Expect the mock test to record a failure when result type is not vector
	if !mockTest.failed {
		t.Error("Expected test to have failed when result type is not vector")
	}
}

func TestCheckNoAlertsFiring_EmptyVectorShouldFail(t *testing.T) {
	t.Parallel()
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
	require.NoError(t, err, "Failed to create client")

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, []string{})

	// Should fail due to query error
	assert.True(t, mockTest.failed, "Test should have failed due to query error")
	assert.NotEmpty(t, mockTest.fatals, "Should have fatal error due to query failure")
}

func TestVMGatherHost(t *testing.T) {
	// Test behavior of VMGatherHost via consts package
	originalHost := consts.NginxHost()
	defer consts.SetNginxHost(originalHost)

	consts.SetNginxHost("192.0.2.1")
	expected := "vmgather.192.0.2.1.nip.io"
	if consts.VMGatherHost() != expected {
		t.Errorf("Expected VMGatherHost to be %s, got %s", expected, consts.VMGatherHost())
	}

	// Empty nginx host should yield empty VMGatherHost
	consts.SetNginxHost("")
	if consts.VMGatherHost() != "" {
		t.Errorf("Expected VMGatherHost to be empty when nginx host is empty, got %s", consts.VMGatherHost())
	}
}
