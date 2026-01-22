package vmcluster_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/gather"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/install"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/promquery"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/remotewrite"
	"github.com/VictoriaMetrics/end-to-end-tests/pkg/tests"
)

func TestVMClusterTests(t *testing.T) {
	tests.Init()
	RegisterFailHandler(Fail)
	suiteConfig, reporterConfig := GinkgoConfiguration()
	RunSpecs(t, "VMCluster test Suite", suiteConfig, reporterConfig)
}

var _ = Describe("VMCluster test", Ordered, ContinueOnFailure, Label("vmcluster"), func() {
	const (
		vmNamespace        = "monitoring"
		overwatchNamespace = "overwatch"
		releaseName        = "vmks"
		helmChart          = "vm/victoria-metrics-k8s-stack"
		valuesFile         = "../../manifests/smoke.yaml"
	)

	ctx := context.Background()
	t := tests.GetT()
	c := &http.Client{
		Timeout: time.Minute,
	}
	var overwatch promquery.PrometheusClient

	BeforeAll(func() {
		install.DiscoverIngressHost(ctx, t)

		var err error
		logger.Default.Logf(t, "Running overwatch at %s", consts.VMSingleUrl())
		overwatch, err = promquery.NewPrometheusClient(fmt.Sprintf("%s/prometheus", consts.VMSingleUrl()))
		require.NoError(t, err)
		overwatch.Start = time.Now()
		install.InstallVMGather(t)

		install.InstallWithHelm(ctx, helmChart, valuesFile, t, vmNamespace, releaseName)
		install.InstallOverwatch(ctx, t, overwatchNamespace, vmNamespace, releaseName)
	})
	AfterEach(func() {
		gather.K8sAfterAll(ctx, t, consts.ResourceWaitTimeout)
		gather.VMAfterAll(ctx, t, consts.ResourceWaitTimeout, releaseName)
	})

	Describe("Multitenancy", func() {
		It("not mix data sent to different tenants", Label("kind", "gke", "id=66618081-b150-4b48-8180-ae1f53512117"), func() {
			By("Inserting data into tenant 0")
			tenantOneInsertURL := fmt.Sprintf("http://%s/insert/0/prometheus/api/v1/write", consts.VMInsertHost(vmNamespace))
			ts := remotewrite.GenTimeSeries("foo", 10, 1)
			err := remotewrite.RemoteWrite(c, ts, tenantOneInsertURL)
			require.NoError(t, err)

			By("Inserting data into tenant 1")
			tenantTwoInsertURL := fmt.Sprintf("http://%s/insert/1/prometheus/api/v1/write", consts.VMInsertHost(vmNamespace))
			ts = remotewrite.GenTimeSeries("bar", 10, 5)
			err = remotewrite.RemoteWrite(c, ts, tenantTwoInsertURL)
			require.NoError(t, err)

			time.Sleep(30 * time.Second)

			By("Verifying data is not mixed")
			tenantOneSelectURL := fmt.Sprintf("%s/select/0/prometheus", consts.VMSelectUrl(vmNamespace))
			tenantOneProm, err := promquery.NewPrometheusClient(tenantOneSelectURL)
			tenantOneProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err := tenantOneProm.VectorValue(ctx, "foo_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(1))
			value, err = tenantOneProm.VectorValue(ctx, "bar_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))

			tenantTwoSelectURL := fmt.Sprintf("%s/select/1/prometheus", consts.VMSelectUrl(vmNamespace))
			tenantTwoProm, err := promquery.NewPrometheusClient(tenantTwoSelectURL)
			tenantTwoProm.Start = overwatch.Start
			require.NoError(t, err)
			value, err = tenantTwoProm.VectorValue(ctx, "bar_2")
			require.NoError(t, err)
			require.Equal(t, value, model.SampleValue(5))
			value, err = tenantTwoProm.VectorValue(ctx, "foo_2")
			require.EqualError(t, err, "no data returned")
			require.Equal(t, value, model.SampleValue(0))
		})
	})
})
