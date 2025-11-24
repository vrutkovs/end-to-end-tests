package end_to_end_tests_test

import (
	"context"
	"os/exec"
	"time"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/stretchr/testify/require"

	vmclient "github.com/VictoriaMetrics/operator/api/client/versioned"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/clientcmd"
	watchtools "k8s.io/client-go/tools/watch"

	. "github.com/onsi/ginkgo/v2"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
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
		helmOpts := &helm.Options{
			KubectlOptions: kubeOpts,
			ValuesFiles:    []string{valuesFile},
			ExtraArgs: map[string][]string{
				"upgrade": {"--create-namespace", "--wait"},
			},
		}

		kubeConfigPath, err := kubeOpts.GetConfigPath(t)
		require.NoError(t, err)
		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath}, &clientcmd.ConfigOverrides{})
		restConfig, err := clientConfig.ClientConfig()
		require.NoError(t, err)
		vmclient := vmclient.NewForConfigOrDie(restConfig)
		require.NoError(t, err)

		overwatch, err := promquery.NewPrometheusClient("http://localhost:8481/select/0/prometheus")
		require.NoError(t, err)

		BeforeAll(func() {
			overwatch.Start = time.Now()

			By("should install vm/victoria-metrics-k8s-stack chart")
			helm.Upgrade(t, helmOpts, "vm/victoria-metrics-k8s-stack", releaseName)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmagent-vmks", consts.Retries, consts.PollingInterval)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmalert-vmks", consts.Retries, consts.PollingInterval)
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vminsert-vmks", consts.Retries, consts.PollingInterval)

			By("should install VMSingle overwatch instance")
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmsingle.yaml")
			k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "vmsingle-overwatch", consts.Retries, consts.PollingInterval)

			By("should reconfigure VMAgent to send data to VMSingle")
			k8s.KubectlApply(t, kubeOpts, "../manifests/overwatch/vmagent.yaml")

			By("should wait for VMCluster object to become operational")
			func() {
				watchInterface, err := vmclient.OperatorV1beta1().VMClusters(namespace).Watch(ctx, metav1.ListOptions{})
				require.NoError(t, err)
				defer watchInterface.Stop()

				timeBoundContext, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
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

				timeBoundContext, cancel := context.WithTimeout(ctx, consts.ResourceWaitTimeout)
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
		})
		AfterEach(func() {
			gather.K8sAfterAll(t, ctx, consts.ResourceWaitTimeout)
			gather.VMAfterAll(t, ctx, consts.ResourceWaitTimeout)
		})

		It("should handle select requests", Label("id=37076a52-94ca-4de1-b1c8-029f8ce56bb7"), func() {
			By("port-forward vmselect address")
			cmd := exec.CommandContext(ctxCancel, "kubectl", "-n", "vm", "port-forward", "svc/vmselect-vmks", "8481:8481")
			go cmd.Run()
			// Hack: give it some time to start
			time.Sleep(1 * time.Second)

			By("send requests for 5 minutes")
			tickerPeriod := time.Second

			promAPI, err := promquery.NewPrometheusClient("http://localhost:8481/select/0/prometheus")
			promAPI.Start = overwatch.Start
			require.NoError(t, err)

			ticker := time.NewTicker(tickerPeriod)
			defer ticker.Stop()

			started := time.Now()
			for ; true; <-ticker.C {
				_, _, err := promAPI.Query(ctx, "up")
				require.NoError(t, err)

				now := <-ticker.C
				if now.Sub(started) > 5*time.Minute {
					break
				}
			}

			// Expect to make at least 40k requests
			By("at least 10k requests were made")
			value, err := overwatch.VectorValue(ctx, "sum(vm_requests_total)")
			require.NoError(t, err)
			require.GreaterOrEqual(t, value, float64(10000))
		})
	})
})
