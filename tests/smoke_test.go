package end_to_end_tests_test

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/gruntwork-io/terratest/modules/k8s"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke test", Label("smoke"), func() {

	Context("k8s-stack", func() {
		const (
			namespace   = "vm"
			releaseName = "vmks"
		)

		t := GetT()
		kubectlOptions := k8s.NewKubectlOptions("", "", namespace)

		// k8sClient, err := k8s.GetKubernetesClientFromOptionsE(t, kubectlOptions)
		// require.NoError(t, err)

		operatorName := fmt.Sprintf("%s-victoria-metrics-operator", releaseName)
		k8s.WaitUntilDeploymentAvailable(t, kubectlOptions, operatorName, retries, pollingInterval)

		It("should handle select request", Label("id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {
			const (
				query = "up"
				step  = "60s"
			)
			reqURL := url.URL{
				Scheme: "http",
				Host:   vmClusterUrl,
				Path:   "/select/0/prometheus/api/v1/query_range",
			}
			q := reqURL.Query()
			q.Add("query", query)
			q.Add("step", step)
			reqURL.RawQuery = q.Encode()

			req, err := http.NewRequest(http.MethodGet, reqURL.String(), nil)
			Expect(err).ToNot(HaveOccurred())

			res, err := http.DefaultClient.Do(req)
			Expect(res).ToNot(BeNil())
			Expect(err).ToNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
