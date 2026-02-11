package promquery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTestingT captures test failures for verification
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

func (m *mockTestingT) Helper() {
	// No-op for mock
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

// Ensure mockTestingT implements the interface required by terratest/testing
var _ interface {
	Name() string
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fail()
	FailNow()
	Failed() bool
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
	Helper()
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	Skip(args ...interface{})
	SkipNow()
	Skipf(format string, args ...interface{})
	Skipped() bool
} = (*mockTestingT)(nil)

func TestCheckNoAlertsFiring_NoAlerts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/query", r.URL.Path)

		err := r.ParseForm()
		require.NoError(t, err)

		query := r.Form.Get("query")
		expectedSubstring := `sum by (alertname) (ALERTS{namespace="ns", alertname!~"InfoInhibitor|Watchdog|RecordingRulesNoData|NodeMemoryMajorPagesFaults", alertstate="firing"})`
		assert.Equal(t, expectedSubstring, query)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
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
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, "ns", []string{})

	assert.False(t, mockTest.failed, "Should not fail when alert value is 0")
}

func TestCheckNoAlertsFiring_WithCustomExceptions(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		require.NoError(t, err)

		query := r.Form.Get("query")
		// Should include custom exception "CustomAlert"
		expectedSubstring := `alertname!~"InfoInhibitor|Watchdog|RecordingRulesNoData|NodeMemoryMajorPagesFaults|CustomAlert"`
		assert.Contains(t, query, expectedSubstring)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, "ns", []string{"CustomAlert"})

	assert.False(t, mockTest.failed)
}

func TestCheckNoAlertsFiring_WithFiringAlerts(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "FiringAlert"},
						"value": [1234567890, "1"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, "ns", []string{})

	assert.True(t, mockTest.failed, "Should fail when alert is firing")
}

func TestCheckNoAlertsFiring_QueryError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	// Should not fail the test on query error (according to implementation)
	client.CheckNoAlertsFiring(ctx, mockTest, "ns", []string{})

	assert.False(t, mockTest.failed, "Should not fail test on query error")
}

func TestCheckNoAlertsFiring_WrongResultType(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": []
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckNoAlertsFiring(ctx, mockTest, "ns", []string{})

	assert.True(t, mockTest.failed, "Should fail on wrong result type")
}

func TestCheckAlertIsFiring_AlertFiring(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "TargetAlert"},
						"value": [1234567890, "1"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "ns", "TargetAlert")

	assert.False(t, mockTest.failed, "Should not fail when target alert is firing")
}

func TestCheckAlertIsFiring_AlertNotFiring(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"alertname": "TargetAlert"},
						"value": [1234567890, "0"]
					}
				]
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "ns", "TargetAlert")

	assert.True(t, mockTest.failed, "Should fail when target alert value is 0")
}

func TestCheckAlertIsFiring_AlertNotFound(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": []
			}
		}`)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	client.CheckAlertIsFiring(ctx, mockTest, "ns", "TargetAlert")

	assert.True(t, mockTest.failed, "Should fail when alert is not found")
}

func TestCheckAlertIsFiring_QueryError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	require.NoError(t, err)

	mockTest := &mockTestingT{}
	ctx := context.Background()

	// The implementation of CheckAlertIsFiring calls require.NoError on query error
	// So it should fail the test
	client.CheckAlertIsFiring(ctx, mockTest, "ns", "TargetAlert")

	assert.True(t, mockTest.failed, "Should fail test on query error")
}
