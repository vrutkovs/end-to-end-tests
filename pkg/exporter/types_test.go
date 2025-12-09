package exporter

import (
	"encoding/json"
	"testing"
	"time"
)

func TestAuthSerialization(t *testing.T) {
	auth := Auth{
		Type: "none",
	}

	data, err := json.Marshal(auth)
	if err != nil {
		t.Fatalf("Failed to marshal Auth: %v", err)
	}

	var unmarshaled Auth
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Auth: %v", err)
	}

	if unmarshaled.Type != auth.Type {
		t.Errorf("Expected Type to be %s, got %s", auth.Type, unmarshaled.Type)
	}
}

func TestConnectionSerialization(t *testing.T) {
	tenantID := 123
	conn := Connection{
		URL:           "http://victoria-metrics:8428",
		APIBasePath:   "/prometheus",
		TenantID:      &tenantID,
		IsMultitenant: true,
		FullAPIURL:    "http://victoria-metrics:8428/prometheus",
		Auth: Auth{
			Type: "basic",
		},
		SkipTLSVerify: true,
	}

	data, err := json.Marshal(conn)
	if err != nil {
		t.Fatalf("Failed to marshal Connection: %v", err)
	}

	var unmarshaled Connection
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Connection: %v", err)
	}

	if unmarshaled.URL != conn.URL {
		t.Errorf("Expected URL to be %s, got %s", conn.URL, unmarshaled.URL)
	}
	if unmarshaled.APIBasePath != conn.APIBasePath {
		t.Errorf("Expected APIBasePath to be %s, got %s", conn.APIBasePath, unmarshaled.APIBasePath)
	}
	if *unmarshaled.TenantID != *conn.TenantID {
		t.Errorf("Expected TenantID to be %d, got %d", *conn.TenantID, *unmarshaled.TenantID)
	}
	if unmarshaled.IsMultitenant != conn.IsMultitenant {
		t.Errorf("Expected IsMultitenant to be %t, got %t", conn.IsMultitenant, unmarshaled.IsMultitenant)
	}
	if unmarshaled.FullAPIURL != conn.FullAPIURL {
		t.Errorf("Expected FullAPIURL to be %s, got %s", conn.FullAPIURL, unmarshaled.FullAPIURL)
	}
	if unmarshaled.Auth.Type != conn.Auth.Type {
		t.Errorf("Expected Auth.Type to be %s, got %s", conn.Auth.Type, unmarshaled.Auth.Type)
	}
	if unmarshaled.SkipTLSVerify != conn.SkipTLSVerify {
		t.Errorf("Expected SkipTLSVerify to be %t, got %t", conn.SkipTLSVerify, unmarshaled.SkipTLSVerify)
	}
}

func TestConnectionWithNullTenantID(t *testing.T) {
	conn := Connection{
		URL:           "http://victoria-metrics:8428",
		APIBasePath:   "/prometheus",
		TenantID:      nil, // Test null value
		IsMultitenant: false,
		FullAPIURL:    "http://victoria-metrics:8428/prometheus",
		Auth: Auth{
			Type: "none",
		},
		SkipTLSVerify: false,
	}

	data, err := json.Marshal(conn)
	if err != nil {
		t.Fatalf("Failed to marshal Connection: %v", err)
	}

	var unmarshaled Connection
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Connection: %v", err)
	}

	if unmarshaled.TenantID != nil {
		t.Errorf("Expected TenantID to be nil, got %v", unmarshaled.TenantID)
	}
}

func TestTimeRangeSerialization(t *testing.T) {
	start := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2023, 12, 1, 11, 0, 0, 0, time.UTC)

	timeRange := TimeRange{
		Start: start,
		End:   end,
	}

	data, err := json.Marshal(timeRange)
	if err != nil {
		t.Fatalf("Failed to marshal TimeRange: %v", err)
	}

	var unmarshaled TimeRange
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal TimeRange: %v", err)
	}

	if !unmarshaled.Start.Equal(timeRange.Start) {
		t.Errorf("Expected Start to be %v, got %v", timeRange.Start, unmarshaled.Start)
	}
	if !unmarshaled.End.Equal(timeRange.End) {
		t.Errorf("Expected End to be %v, got %v", timeRange.End, unmarshaled.End)
	}
}

func TestObfuscationSerialization(t *testing.T) {
	obf := Obfuscation{
		Enabled:           true,
		ObfuscateInstance: false,
		ObfuscateJob:      true,
		PreserveStructure: false,
		CustomLabels:      []string{"label1", "label2", "label3"},
	}

	data, err := json.Marshal(obf)
	if err != nil {
		t.Fatalf("Failed to marshal Obfuscation: %v", err)
	}

	var unmarshaled Obfuscation
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Obfuscation: %v", err)
	}

	if unmarshaled.Enabled != obf.Enabled {
		t.Errorf("Expected Enabled to be %t, got %t", obf.Enabled, unmarshaled.Enabled)
	}
	if unmarshaled.ObfuscateInstance != obf.ObfuscateInstance {
		t.Errorf("Expected ObfuscateInstance to be %t, got %t", obf.ObfuscateInstance, unmarshaled.ObfuscateInstance)
	}
	if unmarshaled.ObfuscateJob != obf.ObfuscateJob {
		t.Errorf("Expected ObfuscateJob to be %t, got %t", obf.ObfuscateJob, unmarshaled.ObfuscateJob)
	}
	if unmarshaled.PreserveStructure != obf.PreserveStructure {
		t.Errorf("Expected PreserveStructure to be %t, got %t", obf.PreserveStructure, unmarshaled.PreserveStructure)
	}
	if len(unmarshaled.CustomLabels) != len(obf.CustomLabels) {
		t.Errorf("Expected CustomLabels length to be %d, got %d", len(obf.CustomLabels), len(unmarshaled.CustomLabels))
	}
	for i, label := range obf.CustomLabels {
		if unmarshaled.CustomLabels[i] != label {
			t.Errorf("Expected CustomLabels[%d] to be %s, got %s", i, label, unmarshaled.CustomLabels[i])
		}
	}
}

func TestBatchingSerialization(t *testing.T) {
	batching := Batching{
		Enabled:            true,
		Strategy:           "custom",
		CustomIntervalSecs: 60,
	}

	data, err := json.Marshal(batching)
	if err != nil {
		t.Fatalf("Failed to marshal Batching: %v", err)
	}

	var unmarshaled Batching
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal Batching: %v", err)
	}

	if unmarshaled.Enabled != batching.Enabled {
		t.Errorf("Expected Enabled to be %t, got %t", batching.Enabled, unmarshaled.Enabled)
	}
	if unmarshaled.Strategy != batching.Strategy {
		t.Errorf("Expected Strategy to be %s, got %s", batching.Strategy, unmarshaled.Strategy)
	}
	if unmarshaled.CustomIntervalSecs != batching.CustomIntervalSecs {
		t.Errorf("Expected CustomIntervalSecs to be %d, got %d", batching.CustomIntervalSecs, unmarshaled.CustomIntervalSecs)
	}
}

func TestRequestBodySerialization(t *testing.T) {
	start := time.Date(2023, 12, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2023, 12, 1, 11, 0, 0, 0, time.UTC)
	tenantID := 42

	reqBody := RequestBody{
		Connection: Connection{
			URL:           "http://test.example.com",
			APIBasePath:   "/api/v1",
			TenantID:      &tenantID,
			IsMultitenant: true,
			FullAPIURL:    "http://test.example.com/api/v1",
			Auth:          Auth{Type: "bearer"},
			SkipTLSVerify: true,
		},
		TimeRange: TimeRange{
			Start: start,
			End:   end,
		},
		Components: []string{"operator", "victoria"},
		Jobs:       []string{"job1", "job2"},
		Obfuscation: Obfuscation{
			Enabled:           false,
			ObfuscateInstance: true,
			ObfuscateJob:      false,
			PreserveStructure: true,
			CustomLabels:      []string{"custom1", "custom2"},
		},
		StagingDir:        "/tmp/staging",
		MetricStepSeconds: 30,
		Batching: Batching{
			Enabled:            true,
			Strategy:           "time",
			CustomIntervalSecs: 120,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal RequestBody: %v", err)
	}

	var unmarshaled RequestBody
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal RequestBody: %v", err)
	}

	// Test Connection
	if unmarshaled.Connection.URL != reqBody.Connection.URL {
		t.Errorf("Expected Connection.URL to be %s, got %s", reqBody.Connection.URL, unmarshaled.Connection.URL)
	}
	if *unmarshaled.Connection.TenantID != *reqBody.Connection.TenantID {
		t.Errorf("Expected Connection.TenantID to be %d, got %d", *reqBody.Connection.TenantID, *unmarshaled.Connection.TenantID)
	}

	// Test TimeRange
	if !unmarshaled.TimeRange.Start.Equal(reqBody.TimeRange.Start) {
		t.Errorf("Expected TimeRange.Start to be %v, got %v", reqBody.TimeRange.Start, unmarshaled.TimeRange.Start)
	}
	if !unmarshaled.TimeRange.End.Equal(reqBody.TimeRange.End) {
		t.Errorf("Expected TimeRange.End to be %v, got %v", reqBody.TimeRange.End, unmarshaled.TimeRange.End)
	}

	// Test Components
	if len(unmarshaled.Components) != len(reqBody.Components) {
		t.Errorf("Expected Components length to be %d, got %d", len(reqBody.Components), len(unmarshaled.Components))
	}
	for i, component := range reqBody.Components {
		if unmarshaled.Components[i] != component {
			t.Errorf("Expected Components[%d] to be %s, got %s", i, component, unmarshaled.Components[i])
		}
	}

	// Test Jobs
	if len(unmarshaled.Jobs) != len(reqBody.Jobs) {
		t.Errorf("Expected Jobs length to be %d, got %d", len(reqBody.Jobs), len(unmarshaled.Jobs))
	}
	for i, job := range reqBody.Jobs {
		if unmarshaled.Jobs[i] != job {
			t.Errorf("Expected Jobs[%d] to be %s, got %s", i, job, unmarshaled.Jobs[i])
		}
	}

	// Test StagingDir
	if unmarshaled.StagingDir != reqBody.StagingDir {
		t.Errorf("Expected StagingDir to be %s, got %s", reqBody.StagingDir, unmarshaled.StagingDir)
	}

	// Test MetricStepSeconds
	if unmarshaled.MetricStepSeconds != reqBody.MetricStepSeconds {
		t.Errorf("Expected MetricStepSeconds to be %d, got %d", reqBody.MetricStepSeconds, unmarshaled.MetricStepSeconds)
	}
}

func TestEmptySlicesSerialization(t *testing.T) {
	reqBody := RequestBody{
		Connection: Connection{
			URL:           "http://test.example.com",
			APIBasePath:   "/api/v1",
			TenantID:      nil,
			IsMultitenant: false,
			FullAPIURL:    "http://test.example.com/api/v1",
			Auth:          Auth{Type: "none"},
			SkipTLSVerify: false,
		},
		TimeRange: TimeRange{
			Start: time.Now().Add(-time.Hour),
			End:   time.Now(),
		},
		Components: []string{}, // Empty slice
		Jobs:       []string{}, // Empty slice
		Obfuscation: Obfuscation{
			Enabled:           false,
			ObfuscateInstance: false,
			ObfuscateJob:      false,
			PreserveStructure: true,
			CustomLabels:      []string{}, // Empty slice
		},
		StagingDir:        "/tmp",
		MetricStepSeconds: 0,
		Batching: Batching{
			Enabled:            false,
			Strategy:           "",
			CustomIntervalSecs: 0,
		},
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("Failed to marshal RequestBody with empty slices: %v", err)
	}

	var unmarshaled RequestBody
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal RequestBody with empty slices: %v", err)
	}

	if len(unmarshaled.Components) != 0 {
		t.Errorf("Expected empty Components slice, got length %d", len(unmarshaled.Components))
	}
	if len(unmarshaled.Jobs) != 0 {
		t.Errorf("Expected empty Jobs slice, got length %d", len(unmarshaled.Jobs))
	}
	if len(unmarshaled.Obfuscation.CustomLabels) != 0 {
		t.Errorf("Expected empty CustomLabels slice, got length %d", len(unmarshaled.Obfuscation.CustomLabels))
	}
}
