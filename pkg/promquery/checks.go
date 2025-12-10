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
	query := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{alertname!~"%s"}) > 0`, strings.Join(allExceptions, "|"))

	result, _, err := p.Query(ctx, query)
	if err != nil {
		require.NoError(t, err, "Failed to query for alerts")
		return
	}

	require.Equal(t, prommodel.ValVector, result.Type())
	vec := result.(prommodel.Vector)
	// At least one result is returned
	require.GreaterOrEqual(t, len(vec), 1, "No alerts firing")
	for _, alert := range vec {
		require.Equal(t, prommodel.SampleValue(0), alert.Value, "Unexpected alert firing: %s", alert.Metric)
	}
}

// CheckAlertIsFiring verifies that a specific alert is currently firing (value > 0)
func (p PrometheusClient) CheckAlertIsFiring(ctx context.Context, t testing.TestingT, alertName string) {
	query := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{alertname="%s"}) > 0`, alertName)

	result, _, err := p.Query(ctx, query)
	if err != nil {
		require.NoError(t, err, "Failed to query for alert %s", alertName)
		return
	}

	require.Equal(t, prommodel.ValVector, result.Type(), "Expected vector result for alert query")
	vec := result.(prommodel.Vector)
	require.Equal(t, len(vec), 1, "Alert %s should be present in results", alertName)
}
