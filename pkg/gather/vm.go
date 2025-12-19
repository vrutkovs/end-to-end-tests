package gather

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ginkgo "github.com/onsi/ginkgo/v2"
	"github.com/stretchr/testify/require"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/exporter"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
)

// VMAfterAll provides cleanup and data collection logic for VictoriaMetrics components.
// It calls vmgather /api/export/start, polls /api/export/status,
// calls /api/export/download endpoints, and adds the downloaded archive to the report.
func VMAfterAll(ctx context.Context, t testing.TestingT, resourceWaitTimeout time.Duration, namespaceList []string) {
	// Set start and end times dynamically
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	// nil for TenantID as per JSON specification
	var tenantID *int = nil

	jobs := []string{"vmagent-vmks", "vmalert-vmks"}
	for _, namespace := range namespaceList {
		jobs = append(jobs, fmt.Sprintf("vmselect-%s", namespace))
		jobs = append(jobs, fmt.Sprintf("vmstorage-%s", namespace))
		jobs = append(jobs, fmt.Sprintf("vminsert-%s", namespace))
	}

	reqBody := exporter.RequestBody{
		Connection: exporter.Connection{
			URL:           fmt.Sprintf("http://%s/prometheus", consts.GetVMSingleSvc("overwatch")),
			APIBasePath:   "/prometheus",
			TenantID:      tenantID,
			IsMultitenant: false,
			FullAPIURL:    fmt.Sprintf("http://%s/prometheus", consts.GetVMSingleSvc("overwatch")),
			Auth:          exporter.Auth{Type: "none"},
			SkipTLSVerify: false,
		},
		TimeRange: exporter.TimeRange{
			Start: startTime,
			End:   endTime,
		},
		Components: []string{"operator", "vmagent", "vmalert", "vminsert", "vmselect", "vmstorage"},
		Jobs:       jobs,
		Obfuscation: exporter.Obfuscation{
			Enabled:           false,
			ObfuscateInstance: false,
			ObfuscateJob:      false,
			PreserveStructure: true,
			CustomLabels:      []string{},
		},
		StagingDir:        "/tmp/staging",
		MetricStepSeconds: 0,
		Batching: exporter.Batching{
			Enabled:            true,
			Strategy:           "custom",
			CustomIntervalSecs: 60,
		},
	}

	marshaledBody, err := json.Marshal(reqBody)
	require.NoError(t, err, "failed to marshal request body")

	// Call vmgather's /api/export/start endpoint
	startURL := url.URL{
		Scheme: "http",
		Host:   consts.VMGatherHost(),
		Path:   "/api/export/start",
	}
	logger.Default.Logf(t, "Making request to %s", startURL.String())
	startReq, err := http.NewRequest(http.MethodPost, startURL.String(), bytes.NewBuffer(marshaledBody))
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
	err = res.Body.Close()
	require.NoError(t, err, "failed to close response body")

	logger.Default.Logf(t, "vmexporter job started with ID: %s", startExportResponse.JobID)

	// Poll for job status until complete
	statusURL := url.URL{
		Scheme: "http",
		Host:   consts.VMGatherHost(),
		Path:   "/api/export/status",
	}
	logger.Default.Logf(t, "Making request to %s", statusURL.String())
	q := statusURL.Query()
	q.Add("id", startExportResponse.JobID)
	statusURL.RawQuery = q.Encode()
	statusURLStr := statusURL.String()

	var archivePath string
	pollCtx, pollCancel := context.WithTimeout(ctx, resourceWaitTimeout)
	defer pollCancel()

OuterLoop:
	for {
		select {
		case <-pollCtx.Done():
			// Exit loop if context is cancelled, check archivePath later
			break OuterLoop
		default:
			statusReq, err := http.NewRequest(http.MethodGet, statusURLStr, nil)
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
			err = statusRes.Body.Close()
			require.NoError(t, err, "failed to close response body")

			logger.Default.Logf(t, "vmexporter job %s status: %s", startExportResponse.JobID, statusResponse.State)

			switch statusResponse.State {
			case "completed":
				archivePath = statusResponse.Result.ArchivePath
				require.NotEmpty(t, archivePath, "archive_path should not be empty when state is complete")
				break OuterLoop
			case "failed":
				logger.Default.Logf(t, "vmexporter job %s statusResponse: %#v", startExportResponse.JobID, statusResponse)
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
		Host:   consts.VMGatherHost(),
		Path:   "/api/download",
	}
	q = downloadURL.Query()
	q.Add("path", archivePath)
	downloadURL.RawQuery = q.Encode()
	downloadURLStr := downloadURL.String()

	req, err := http.NewRequest(http.MethodGet, downloadURLStr, nil)
	require.NoError(t, err, "failed to create HTTP request for /api/download")

	res, err = http.DefaultClient.Do(req)
	require.NoError(t, err, "failed to perform HTTP request to /api/download")
	require.Equal(t, http.StatusOK, res.StatusCode, "unexpected status code from /api/download")

	// Store the downloaded zip file content in a buffer
	var zipBuffer bytes.Buffer
	_, err = zipBuffer.ReadFrom(res.Body)
	require.NoError(t, err, "failed to read downloaded zip to buffer")
	err = res.Body.Close()
	require.NoError(t, err, "failed to close response body")

	logger.Default.Logf(t, "Downloaded vmexporter archive into buffer, size: %d bytes", zipBuffer.Len())

	// Add the downloaded zip file content to the report
	ginkgo.AddReportEntry("vmexporter-report.zip", zipBuffer.String(), ginkgo.ReportEntryVisibilityNever)
}

func RestartOverwatchInstance(ctx context.Context, t testing.TestingT, namespace string) {
	kubeOpts := k8s.NewKubectlOptions("", "", namespace)
	client, err := k8s.GetKubernetesClientE(t)
	require.NoError(t, err, "failed to get Kubernetes client")

	pods := k8s.ListPods(t, kubeOpts, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/instance=overwatch",
	})
	firstPod := pods[0]

	// TODO: Implement logic to restart overwatch instance
	err = client.CoreV1().Pods(namespace).Delete(ctx, firstPod.Name, metav1.DeleteOptions{})
	require.NoError(t, err, "failed to delete pod %s", firstPod.Name)

	// Wait for overwatch VMSingle to become operational
	vmclient := install.GetVMClient(t, kubeOpts)
	install.WaitForVMSingleToBeOperational(ctx, t, kubeOpts, namespace, vmclient)
}
