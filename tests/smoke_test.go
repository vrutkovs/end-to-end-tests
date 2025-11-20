package end_to_end_tests_test

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"

	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"

	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke test", Ordered, Label("smoke"), func() {

	Context("k8s-stack", func() {
		const (
			namespace   = "vm"
			releaseName = "vmks"
			valuesFile  = "../manifests/smoke.yaml"
		)

		ctx := context.Background()
		t := GetT()
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)
		helmOpts := &helm.Options{
			BuildDependencies: true,
			KubectlOptions:    kubeOpts,
			ValuesFiles:       []string{valuesFile},
		}

		k8sClient, err := k8s.GetKubernetesClientFromOptionsE(t, kubeOpts)
		require.NoError(t, err)

		k8s.CreateNamespace(t, kubeOpts, "vm")

		AfterAll(func() {
			helm.Delete(t, helmOpts, releaseName, true)
		})

		It("should install vm/victoria-metrics-k8s-stack chart", func() {
			helm.Upgrade(t, helmOpts, "vm/victoria-metrics-k8s-stack", releaseName)
		})

		It("should install VMSingle overwatch instance", func() {
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmsingle.yaml")
		})

		It("should reconfigure VMAgent to send data to VMSingle", func() {
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmagent.yaml")
		})

		It("should wait for VMCluster object to become operational", func() {
			watchInterface, err := k8sClient.RESTClient().Get().Resource("vmclusters").Namespace(namespace).Name("vmcluster").Watch(ctx)
			require.NoError(t, err)
			defer watchInterface.Stop()

			timeBoundContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			_, err = watchtools.UntilWithoutRetry(timeBoundContext, watchInterface, func(event watch.Event) (bool, error) {
				obj := event.Object
				vmCluster := obj.(*vmv1beta1.VMCluster)
				if vmCluster.Status.UpdateStatus == vmv1beta1.UpdateStatusOperational {
					return true, nil
				}
				return false, nil
			})
			require.NoError(t, err)
		})

		It("should wait for overwatch VMSingle to become operational", func() {
			watchInterface, err := k8sClient.RESTClient().Get().Resource("vmsingles").Namespace(namespace).Name("vmsingle").Watch(ctx)
			require.NoError(t, err)
			defer watchInterface.Stop()

			timeBoundContext, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			_, err = watchtools.UntilWithoutRetry(timeBoundContext, watchInterface, func(event watch.Event) (bool, error) {
				obj := event.Object
				vmSingle := obj.(*vmv1beta1.VMSingle)
				if vmSingle.Status.UpdateStatus == vmv1beta1.UpdateStatusOperational {
					return true, nil
				}
				return false, nil
			})
			require.NoError(t, err)
		})

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
