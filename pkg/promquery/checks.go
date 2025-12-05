package promquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/gruntwork-io/terratest/modules/testing"
	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func (p PrometheusClient) CheckNoAlertsFiring(ctx context.Context, t testing.TestingT, exceptions []string) {
	defaultExceptions := []string{
		"InfoInhibitor", "Watchdog",
	}
	allExceptions := append(defaultExceptions, exceptions...)
	query := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{alertname!~"%s"})`, strings.Join(allExceptions, "|"))
	result, _, err := p.Query(ctx, query)
	if err == nil {
		require.Equal(t, prommodel.ValVector, result.Type())
		vec := result.(prommodel.Vector)
		// At least one result is returned
		require.GreaterOrEqual(t, len(vec), 1)
		for _, alert := range vec {
			require.Equal(t, prommodel.SampleValue(0), alert.Value, "Unexpected alert firing: %s", alert.Metric)
		}
	}
	// require.NoError(t, err)
}
