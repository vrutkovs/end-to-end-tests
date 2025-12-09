package install

import (
	"context"
	"fmt"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
)

func DiscoverIngressHost(ctx context.Context, t terratesting.TestingT) {
	kubeOpts := k8s.NewKubectlOptions("", "", "ingress-nginx")

	k8s.WaitUntilDeploymentAvailable(t, kubeOpts, "ingress-nginx-controller", consts.Retries, consts.PollingInterval)

	// Fetch the ingress host from the ingress controller service status
	svc := k8s.GetService(t, kubeOpts, "ingress-nginx-controller")
	if svc == nil {
		t.Fatalf("failed to get ingress-nginx-controller service")
		return
	}
	logger.Default.Logf(t, "nginx controller service status: %v", svc.Status)

	var nginxHost string
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		if consts.EnvK8SDistro() != "kind" {
			t.Fatalf("failed to get ingress-nginx-controller service IP")
			return
		}
		nginxHost = "127.0.0.1"
	} else {
		nginxHost = svc.Status.LoadBalancer.Ingress[0].IP
	}
	logger.Default.Logf(t, "nginxHost host: %s", nginxHost)

	consts.SetVMSelectHost(fmt.Sprintf("%s.%s.nip.io", "vmselect", nginxHost))
	consts.SetVMSingleHost(fmt.Sprintf("%s.%s.nip.io", "vmsingle", nginxHost))
	consts.SetVMSelectUrl(fmt.Sprintf("http://%s", consts.VMSelectHost()))
	consts.SetVMSingleUrl(fmt.Sprintf("http://%s", consts.VMSingleHost()))
}
