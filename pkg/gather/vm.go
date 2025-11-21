package gather

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/exporter"
)

// VMAfterAll provides cleanup and data collection logic for VictoriaMetrics components.
// It starts vmexporter, calls its /api/export/start, polls /api/export/status,
// calls /api/export/download endpoints, and adds the downloaded archive to the report.
func VMAfterAll(t testing.TestingT, ctx context.Context, resourceWaitTimeout time.Duration) {
	timeBoundContext, cancel := context.WithTimeout(ctx, resourceWaitTimeout)
	defer cancel()

	// Port-forward vmsingle-overwatch service
	portForwardCmd := exec.CommandContext(timeBoundContext, "kubectl", "-n", "vm", "port-forward", "svc/vmsingle-overwatch", "8429:8429")
	go portForwardCmd.Run()
	// Hack: give it some time to start
	time.Sleep(1 * time.Second)

	// Start vmexporter binary with -no-browser in goroutine
	vmexporterCmd := exec.CommandContext(timeBoundContext, "vmexporter", "-no-browser")
	var vmexporterOutb, vmexporterErrb bytes.Buffer
	vmexporterCmd.Stdout = &vmexporterOutb
	vmexporterCmd.Stderr = &vmexporterErrb
	go func() {
		err := vmexporterCmd.Run()
		if err != nil && err.Error() != "signal: killed" { // Ignore killed signal
			logger.Default.Logf(t, "vmexporter exited with error: %v, stdout: %s, stderr: %s", err, vmexporterOutb.String(), vmexporterErrb.String())
		}
	}()
	// Give vmexporter some time to start
	time.Sleep(2 * time.Second)

	// Prepare the request body using the exporter.RequestBody struct
	// Set start and end times dynamically
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// nil for TenantID as per JSON specification
	var tenantID *int = nil

	reqBody := exporter.RequestBody{
		Connection: exporter.Connection{
			URL:           "http://localhost:8429",
			APIBasePath:   "/prometheus",
			TenantID:      tenantID,
			IsMultitenant: false,
			FullAPIURL:    "http://localhost:8429/prometheus",
			Auth:          exporter.Auth{Type: "none"},
			SkipTLSVerify: false,
		},
		TimeRange: exporter.TimeRange{
			Start: startTime,
			End:   endTime,
		},
		Components: []string{"operator", "victoria", "vmagent", "vmalert", "vminsert", "vmselect", "vmstorage"},
		Jobs:       []string{"vmks-victoria-metrics-operator", "vmsingle-overwatch", "vmagent-vmks", "vmalert-vmks", "vminsert-vmks", "vmselect-vmks", "vmstorage-vmks"},
		Obfuscation: exporter.Obfuscation{
			Enabled:           false,
			ObfuscateInstance: false,
			ObfuscateJob:      false,
			PreserveStructure: true,
			CustomLabels:      []string{},
		},
		StagingDir:        "/tmp/staging", // Use /tmp/staging as specified in the request
		MetricStepSeconds: 0,
		Batching: exporter.Batching{
			Enabled:            true,
			Strategy:           "custom",
			CustomIntervalSecs: 60,
		},
	}

	marshaledBody, err := json.Marshal(reqBody)
	require.NoError(t, err, "failed to marshal request body")

	// Call localhost:8080/api/export/start endpoint with JSON body
	exportStartURL := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/api/export/start",
	}
	startReq, err := http.NewRequest(http.MethodPost, exportStartURL.String(), bytes.NewBuffer(marshaledBody))
	require.NoError(t, err, "failed to create HTTP request for /api/export/start")
	startReq.Header.Set("Content-Type", "application/json")
	logger.Default.Logf(t, "vmexporter /api/export/start request body: %s", string(marshaledBody))

	res, err := http.DefaultClient.Do(startReq)
	require.NoError(t, err, "failed to perform HTTP request to /api/export/start")
	require.Equal(t, http.StatusOK, res.StatusCode, "unexpected status code from /api/export/start")

	var startExportResponse struct {
		JobID string `json:"job_id"`
	}
	err = json.NewDecoder(res.Body).Decode(&startExportResponse)
	require.NoError(t, err, "failed to decode response from /api/export/start")
	require.NotEmpty(t, startExportResponse.JobID, "job_id should not be empty in /api/export/start response")
	res.Body.Close()

	logger.Default.Logf(t, "vmexporter job started with ID: %s", startExportResponse.JobID)

	// Poll for job status until complete
	statusURL := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/api/export/status",
	}
	q := statusURL.Query()
	q.Add("id", startExportResponse.JobID)
	statusURL.RawQuery = q.Encode()

	var archivePath string
	pollCtx, pollCancel := context.WithTimeout(ctx, resourceWaitTimeout)
	defer pollCancel()

OuterLoop:
	for {
		select {
		case <-pollCtx.Done():
			// Exit loop if context is cancelled, check archivePath later
			break
		default:
			statusReq, err := http.NewRequest(http.MethodGet, statusURL.String(), nil)
			require.NoError(t, err, "failed to create HTTP request for /api/export/status")

			statusRes, err := http.DefaultClient.Do(statusReq)
			require.NoError(t, err, "failed to perform HTTP request to /api/export/status")
			require.Equal(t, http.StatusOK, statusRes.StatusCode, "unexpected status code from /api/export/status")

			var statusResponse struct {
				State  string `json:"state"`
				Result struct {
					ArchivePath string `json:"archive_path"`
				} `json:"result"`
			}
			err = json.NewDecoder(statusRes.Body).Decode(&statusResponse)
			require.NoError(t, err, "failed to decode response from /api/export/status")
			statusRes.Body.Close()

			logger.Default.Logf(t, "vmexporter job %s status: %s", startExportResponse.JobID, statusResponse.State)

			switch statusResponse.State {
			case "completed":
				archivePath = statusResponse.Result.ArchivePath
				require.NotEmpty(t, archivePath, "archive_path should not be empty when state is complete")
				break OuterLoop
			case "failed":
				require.FailNow(t, fmt.Sprintf("vmexporter job %s failed", startExportResponse.JobID))
			default:
				time.Sleep(5 * time.Second)
			}
		}
	}

	// Check if archivePath was obtained, if not, it means polling timed out
	if archivePath == "" {
		require.FailNow(t, "polling for export status timed out without completion or failure")
	}

	logger.Default.Logf(t, "vmexporter job %s completed, archive path: %s", startExportResponse.JobID, archivePath)

	// Call download endpoint with archive_path as query param
	downloadURL := url.URL{
		Scheme: "http",
		Host:   "localhost:8080",
		Path:   "/api/download",
	}
	q = downloadURL.Query() // Reuse q from statusURL for building download query
	q.Add("path", archivePath)
	downloadURL.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, downloadURL.String(), nil)
	require.NoError(t, err, "failed to create HTTP request for /api/download")

	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to perform HTTP request to /api/download")
	require.Equal(t, http.StatusOK, res.StatusCode, "unexpected status code from /api/download")

	// Store the downloaded zip file content in a buffer
	var zipBuffer bytes.Buffer
	_, err = zipBuffer.ReadFrom(res.Body)
	require.NoError(t, err, "failed to read downloaded zip to buffer")
	res.Body.Close()

	logger.Default.Logf(t, "Downloaded vmexporter archive into buffer, size: %d bytes", zipBuffer.Len())

	// Add the downloaded zip file content to the report
	ginkgo.AddReportEntry("vmexporter-report.zip", string(zipBuffer.Bytes()))

	// Shutdown vmexporter command
	if vmexporterCmd.Process != nil {
		vmexporterCmd.Process.Kill()
	}

	// Shutdown port-forward command
	if portForwardCmd.Process != nil {
		portForwardCmd.Process.Kill()
	}
}
