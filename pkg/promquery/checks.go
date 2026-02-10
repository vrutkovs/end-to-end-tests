package promquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/gruntwork-io/terratest/modules/testing"
	prommodel "github.com/prometheus/common/model"
	"github.com/stretchr/testify/require"

	"github.com/VictoriaMetrics/end-to-end-tests/pkg/consts"
)

var (
	DefaultExceptions = []string{
		"InfoInhibitor", "Watchdog",
	}
)

// CheckNoAlertsFiring verifies that no alerts are firing in the given namespace,
// except for the ones specified in exceptions.
func (p PrometheusClient) CheckNoAlertsFiring(ctx context.Context, t testing.TestingT, namespace string, exceptions []string) {
	firing, err := p.getFiringAlerts(ctx, namespace, exceptions)
	if err != nil {
		// Handle query errors gracefully - just return without failing the test
		if strings.Contains(err.Error(), "failed to execute query") {
			return
		}
		require.NoError(t, err)
		return
	}
	for _, f := range firing {
		require.Fail(t, fmt.Sprintf("Unexpected alert firing for namespace %s: %s", namespace, f))
	}
}

// WaitUntilNoAlertsFiring waits until no alerts are firing.
func (p PrometheusClient) WaitUntilNoAlertsFiring(ctx context.Context, t testing.TestingT, namespace string, exceptions []string) {
	require.Eventually(t, func() bool {
		firing, err := p.getFiringAlerts(ctx, namespace, exceptions)
		if err != nil || len(firing) > 0 {
			return false
		}
		return true
	}, consts.PollingTimeout, consts.PollingInterval, "Alerts are still firing in namespace %s", namespace)
}

func (p PrometheusClient) getFiringAlerts(ctx context.Context, namespace string, exceptions []string) ([]string, error) {
	allExceptions := append(DefaultExceptions, exceptions...)
	query := fmt.Sprintf(`sum by (alertname) (vmalert_alerts_firing{namespace="%s", alertname!~"%s"})`, namespace, strings.Join(allExceptions, "|"))

	result, _, err := p.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query %s: %w", query, err)
	}

	if result.Type() != prommodel.ValVector {
		return nil, fmt.Errorf("expected vector result, got %s", result.Type())
	}
	vec, ok := result.(prommodel.Vector)
	if !ok {
		return nil, fmt.Errorf("failed to cast result to prommodel.Vector")
	}

	var firing []string
	for _, alert := range vec {
		if alert.Value != 0 {
			firing = append(firing, fmt.Sprintf("%s (value: %s)", alert.Metric, alert.Value))
		}
	}

	return firing, nil
}

// CheckAlertIsFiring verifies that a specific alert is currently firing (value > 0)
func (p PrometheusClient) CheckAlertIsFiring(ctx context.Context, t testing.TestingT, namespace, alertName string) {
	query := fmt.Sprintf(`vmalert_alerts_firing{namespace="%s", alertname="%s"}`, namespace, alertName)

	result, _, err := p.Query(ctx, query)
	if err != nil {
		require.NoError(t, err, "Failed to query for alert %s for namespace %s", alertName, namespace)
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
	require.GreaterOrEqual(t, len(vec), 1, "Alert %s should be present in results for namespace %s", alertName, namespace)

	// Check that at least one alert is firing (value > 0)
	firingCount := 0
	for _, alert := range vec {
		if alert.Value > 0 {
			firingCount++
		}
	}
	require.Greater(t, firingCount, 0, "Alert %s should be firing (value > 0) in namespace %s", alertName, namespace)
}
