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
	require.NoError(t, err)
	require.Equal(t, prommodel.ValVector, result.Type())
	vec := result.(prommodel.Vector)
	require.Len(t, vec, 1)
	require.Equal(t, 0.0, vec[0].Value)
}
