package end_to_end_tests_test

import (
	"context"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"

	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
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

		ctxCancel, cancel := context.WithCancel(ctx)
		AfterAll(func() {
			cancel()
		})

		t := GetT()
		kubeOpts := k8s.NewKubectlOptions("", "", namespace)
		// helmOpts := &helm.Options{
		// 	KubectlOptions: kubeOpts,
		// 	ValuesFiles:    []string{valuesFile},
		// 	ExtraArgs: map[string][]string{
		// 		"upgrade": {"--create-namespace", "--wait"},
		// 	},
		// }

		kubeConfigPath, err := kubeOpts.GetConfigPath(t)
		require.NoError(t, err)
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}, &clientcmd.ConfigOverrides{})
		restConfig, err := clientConfig.ClientConfig()
		require.NoError(t, err)
		vmclient := vmclient.NewForConfigOrDie(restConfig)
		require.NoError(t, err)

		It("should install the stack from the chart", Label("id=69ec6c61-f40d-4c48-ad1f-d60ab5988ee6"), func() {
			By("should install vm/victoria-metrics-k8s-stack chart")
			// helm.Upgrade(t, helmOpts, "vm/victoria-metrics-k8s-stack", releaseName)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmagent-vmks", retries, pollingInterval)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmalert-vmks", retries, pollingInterval)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vminsert-vmks", retries, pollingInterval)

			By("should install VMSingle overwatch instance")
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmsingle.yaml")
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmsingle-overwatch", retries, pollingInterval)

			By("should reconfigure VMAgent to send data to VMSingle")
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmagent.yaml")

			By("should wait for VMCluster object to become operational")
			func() {
				watchInterface, err := vmclient.OperatorV1beta1().VMClusters(namespace).Watch(ctx, metav1.ListOptions{})
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
			}()

			By("should wait for overwatch VMSingle to become operational")
			func() {
				watchInterface, err := vmclient.OperatorV1beta1().VMSingles(namespace).Watch(ctx, metav1.ListOptions{})
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
			}()

			By("should port-forward vmselect address")
			cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmselect-vmks", "8481:8481")
			go cmd.Run()
			time.Sleep(1 * time.Second)
		})

		It("should handle select request", Label("id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {
			const (
				query = "up"
				step  = "60s"
			)
			reqURL := url.URL{
				Scheme: "http",
				Host:   "localhost:8481",
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
