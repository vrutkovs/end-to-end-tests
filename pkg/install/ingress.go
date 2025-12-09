package install

import (
	"context"
	"fmt"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/gruntwork-io/terratest/modules/k8s"
	terratesting "github.com/gruntwork-io/terratest/modules/testing"
)

func DiscoverIngressHost(ctx context.Context, t terratesting.TestingT) {
	kubeOpts := k8s.NewKubectlOptions("", "", "ingress-nginx")

	// Fetch the ingress host from the ingress controller service status
	k8s.WaitUntilServiceAvailable(t, kubeOpts, "ingress-nginx-controller", consts.Retries, consts.PollingInterval)
	svc := k8s.GetService(t, kubeOpts, "ingress-nginx-controller")
	if svc == nil || len(svc.Status.LoadBalancer.Ingress) == 0 {
		t.Fatalf("failed to get ingress-nginx-controller service")
	}
	nginxHost := svc.Status.LoadBalancer.Ingress[0].IP

	consts.VMSelectHost = fmt.Sprintf("%s.%s.nip.io", "vmselect", nginxHost)
	consts.VMSingleHost = fmt.Sprintf("%s.%s.nip.io", "vmsingle", nginxHost)
	consts.VMSelectUrl = fmt.Sprintf("http://%s", consts.VMSelectHost)
	consts.VMSingleUrl = fmt.Sprintf("http://%s", consts.VMSingleHost)
}
