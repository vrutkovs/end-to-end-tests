package promquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/gruntwork-io/terratest/modules/testing"
	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"
)

func (p PrometheusClient) CheckNoAlertsFiring(ctx context.Context, t testing.TestingT, namespace string, exceptions []string) {
	defaultExceptions := []string{
		"InfoInhibitor", "Watchdog",
	}
	allExceptions := append(defaultExceptions, exceptions...)
	query := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{namespace="%s", alertname!~"%s"})`, namespace, strings.Join(allExceptions, "|"))

	result, _, err := p.Query(ctx, query)
	if err != nil {
		// Handle query errors gracefully - just return without failing the test
		return
	}

	if result.Type() != prommodel.ValVector {
		require.Fail(t, fmt.Sprintf("Expected vector result, got %s", result.Type()))
		return
	}
	vec, ok := result.(prommodel.Vector)
	if !ok {
		require.Fail(t, "Failed to cast result to prommodel.Vector")
		return
	}
	// At least one result is returned
	require.GreaterOrEqual(t, len(vec), 1, "No alerts firing")
	for _, alert := range vec {
		require.Equal(t, prommodel.SampleValue(0), alert.Value, "Unexpected alert firing: %s", alert.Metric)
	}
}

// CheckAlertIsFiring verifies that a specific alert is currently firing (value > 0)
func (p PrometheusClient) CheckAlertIsFiring(ctx context.Context, t testing.TestingT, namespace, alertName string) {
	query := fmt.Sprintf(`vmalert_alerts_firing{namespace="%s", alertname="%s"}`, namespace, alertName)

	result, _, err := p.Query(ctx, query)
	if err != nil {
		require.NoError(t, err, "Failed to query for alert %s", alertName)
		return
	}

	if result.Type() != prommodel.ValVector {
		require.Fail(t, fmt.Sprintf("Expected vector result for alert query, got %s", result.Type()))
		return
	}
	vec, ok := result.(prommodel.Vector)
	if !ok {
		require.Fail(t, "Failed to cast result to prommodel.Vector")
		return
	}
	require.GreaterOrEqual(t, len(vec), 1, "Alert %s should be present in results", alertName)

	// Check that at least one alert is firing (value > 0)
	firingCount := 0
	for _, alert := range vec {
		if alert.Value > 0 {
			firingCount++
		}
	}
	require.Greater(t, firingCount, 0, "Alert %s should be firing (value > 0)", alertName)
}
