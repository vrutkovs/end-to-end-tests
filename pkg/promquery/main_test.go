package promquery

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	prommodel "github.com/prometheus/common/model"
)

func TestNewPrometheusClient(t *testing.T) {
	t.Parallel()
	// Test valid URL
	client, err := NewPrometheusClient("http://localhost:9090")
	if err != nil {
		t.Fatalf("Expected no error creating client, got: %v", err)
	}
	if client.client == nil {
		t.Fatal("Expected client to be initialized")
	}

	// Test invalid URL
	_, err = NewPrometheusClient("://invalid-url")
	if err == nil {
		t.Fatal("Expected error for invalid URL")
	}
}

func TestVectorValue(t *testing.T) {
	t.Parallel()
	// Create a mock server that returns different types of responses
	tests := []struct {
		name          string
		response      string
		expectedValue prommodel.SampleValue
		expectedError bool
		errorContains string
	}{
		{
			name: "valid vector response",
			response: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": [
						{
							"metric": {"__name__": "test_metric"},
							"value": [1234567890, "42.5"]
						}
					]
				}
			}`,
			expectedValue: 42.5,
			expectedError: false,
		},
		{
			name: "empty vector response",
			response: `{
				"status": "success",
				"data": {
					"resultType": "vector",
					"result": []
				}
			}`,
			expectedValue: 0,
			expectedError: true,
			errorContains: "no data returned",
		},
		{
			name: "matrix response (wrong type)",
			response: `{
				"status": "success",
				"data": {
					"resultType": "matrix",
					"result": []
				}
			}`,
			expectedValue: 0,
			expectedError: true,
			errorContains: "unexpected result type: matrix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				fmt.Fprint(w, tt.response)
			}))
			defer server.Close()

			// Create client
			client, err := NewPrometheusClient(server.URL)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Test VectorValue
			value, err := client.VectorValue(context.Background(), "test_query")

			if tt.expectedError {
				if err == nil {
					t.Fatalf("Expected error but got none")
				}
				if tt.errorContains != "" && err.Error() != tt.errorContains {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if value != tt.expectedValue {
				t.Errorf("Expected value %v, got %v", tt.expectedValue, value)
			}
		})
	}
}

func TestQuery(t *testing.T) {
	t.Parallel()
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request parameters
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

		query := r.Form.Get("query")
		if query != "test_metric" {
			t.Errorf("Expected query 'test_metric', got '%s'", query)
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "vector",
				"result": [
					{
						"metric": {"__name__": "test_metric"},
						"value": [1234567890, "10"]
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

	ctx := context.Background()
	result, warnings, err := client.Query(ctx, "test_metric")

	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(warnings) > 0 {
		t.Errorf("Unexpected warnings: %v", warnings)
	}

	if result.Type() != prommodel.ValVector {
		t.Errorf("Expected vector result, got %s", result.Type())
	}
}

func TestQueryRange(t *testing.T) {
	t.Parallel()
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request parameters
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/api/v1/query_range" {
			t.Errorf("Expected path '/api/v1/query_range', got '%s'", r.URL.Path)
		}

		// Parse form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("Failed to parse form: %v", err)
			return
		}

		query := r.Form.Get("query")
		if query != "test_range_metric" {
			t.Errorf("Expected query 'test_range_metric', got '%s'", query)
		}

		// Check that start, end, and step parameters are present
		if r.Form.Get("start") == "" {
			t.Error("Expected start parameter")
		}
		if r.Form.Get("end") == "" {
			t.Error("Expected end parameter")
		}
		if r.Form.Get("step") == "" {
			t.Error("Expected step parameter")
		}

		// Return a mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"status": "success",
			"data": {
				"resultType": "matrix",
				"result": [
					{
						"metric": {"__name__": "test_range_metric"},
						"values": [
							[1234567890, "10"],
							[1234567920, "20"]
						]
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

	// Set start time for the client
	client.Start = time.Now().Add(-1 * time.Hour)

	ctx := context.Background()
	result, warnings, err := client.QueryRange(ctx, "test_range_metric")

	if err != nil {
		t.Fatalf("QueryRange failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if len(warnings) > 0 {
		t.Errorf("Unexpected warnings: %v", warnings)
	}

	if result.Type() != prommodel.ValMatrix {
		t.Errorf("Expected matrix result, got %s", result.Type())
	}
}

func TestQueryWithTimeout(t *testing.T) {
	t.Parallel()
	// Create a slow server that takes longer than a custom short timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Short delay for testing
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Use a custom timeout that's shorter than the server delay
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err = client.Query(ctx, "slow_query")

	if err == nil {
		t.Fatal("Expected timeout error")
	}

	// The error should contain context deadline exceeded
	if !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Errorf("Expected context deadline exceeded error, got: %v", err)
	}
}

func TestQueryRangeWithTimeout(t *testing.T) {
	t.Parallel()
	// Create a slow server that takes longer than a custom short timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond) // Short delay for testing
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewPrometheusClient(server.URL)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set start time for the client
	client.Start = time.Now().Add(-1 * time.Hour)

	// Use a custom timeout that's shorter than the server delay
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, _, err = client.QueryRange(ctx, "slow_range_query")

	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

func TestClientStartTime(t *testing.T) {
	t.Parallel()
	client, err := NewPrometheusClient("http://localhost:9090")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test that Start time can be set and retrieved
	testTime := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
	client.Start = testTime

	if !client.Start.Equal(testTime) {
		t.Errorf("Expected Start time to be %v, got %v", testTime, client.Start)
	}
}
