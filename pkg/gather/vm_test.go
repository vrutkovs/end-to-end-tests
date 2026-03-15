package gather

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type roundTripFunc func(req *http.Request) *http.Response

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

var _ = Describe("VM Gather", func() {
	var (
		ctx            context.Context
		originalClient *http.Client
		originalHost   string
	)

	BeforeEach(func() {
		ctx = context.Background()
		originalClient = http.DefaultClient
		http.DefaultClient = &http.Client{}
		originalHost = consts.NginxHost()
		consts.SetNginxHost("test-host")
	})

	AfterEach(func() {
		http.DefaultClient = originalClient
		consts.SetNginxHost(originalHost)
	})

	Context("VMAfterAll", func() {
		It("should successfully complete the export process", func() {
			mockT := &mockTestingT{}
			http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) *http.Response {
				switch req.URL.Path {
				case "/api/export/start":
					body := `{"job_id": "test-job"}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				case "/api/export/status":
					body := `{"state": "completed", "result": {"archive_path": "/tmp/test.zip"}}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				case "/api/download":
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString("fake-zip-content")),
						Header:     make(http.Header),
					}
				default:
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
						Header:     make(http.Header),
					}
				}
			})

			VMAfterAll(ctx, mockT, 5*time.Second, "test-release")
			Expect(mockT.failed).To(BeFalse())
		})

		It("should handle failed job status", func() {
			mockT := &mockTestingT{}
			http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) *http.Response {
				switch req.URL.Path {
				case "/api/export/start":
					body := `{"job_id": "test-job"}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				case "/api/export/status":
					body := `{"state": "failed"}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				default:
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
						Header:     make(http.Header),
					}
				}
			})

			VMAfterAll(ctx, mockT, 5*time.Second, "test-release")
			Expect(mockT.failed).To(BeFalse())
		})

		It("should handle start endpoint failure", func() {
			mockT := &mockTestingT{}
			http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) *http.Response {
				return &http.Response{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(bytes.NewBufferString("error")),
					Header:     make(http.Header),
				}
			})

			VMAfterAll(ctx, mockT, 5*time.Second, "test-release")
			Expect(mockT.failed).To(BeFalse())
		})

		It("should handle context timeout during polling", func() {
			mockT := &mockTestingT{}
			http.DefaultClient.Transport = roundTripFunc(func(req *http.Request) *http.Response {
				switch req.URL.Path {
				case "/api/export/start":
					body := `{"job_id": "test-job"}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				case "/api/export/status":
					body := `{"state": "running"}`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(body)),
						Header:     make(http.Header),
					}
				default:
					return &http.Response{
						StatusCode: http.StatusNotFound,
						Body:       io.NopCloser(bytes.NewBufferString("not found")),
						Header:     make(http.Header),
					}
				}
			})

			// Set a very short timeout
			VMAfterAll(ctx, mockT, 100*time.Millisecond, "test-release")
			Expect(mockT.failed).To(BeFalse())
		})
	})

	Context("RestartOverwatchInstance", func() {
		It("should fail gracefully when cluster is not available", func() {
			mockT := &mockTestingT{}
			kubeOpts := &k8s.KubectlOptions{
				ConfigPath: "/non-existent-path",
			}

			done := make(chan struct{})
			go func() {
				defer close(done)
				RestartOverwatchInstance(ctx, mockT, kubeOpts, "test-ns")
			}()

			<-done
			Expect(mockT.failed).To(BeTrue())
		})
	})
})
