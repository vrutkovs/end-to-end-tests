package promquery

import (
	"context"
	"fmt"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
)

const (
	queryTimeout = 10 * time.Second
	queryStep    = 1 * time.Minute
)

type PrometheusClient struct {
	client promv1.API
	Start  time.Time
}

func NewPrometheusClient(url string) (PrometheusClient, error) {
	promClient, err := promapi.NewClient(promapi.Config{
		Address: url,
	})
	if err != nil {
		return PrometheusClient{}, err
	}
	promv1api := promv1.NewAPI(promClient)
	return PrometheusClient{client: promv1api}, nil
}

func (p PrometheusClient) QueryRange(ctx context.Context, query string) (prommodel.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	return p.client.QueryRange(ctx, query, promv1.Range{
		Start: p.Start,
		End:   time.Now(),
		Step:  queryStep,
	})
}

func (p PrometheusClient) Query(ctx context.Context, query string) (prommodel.Value, promv1.Warnings, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	return p.client.Query(ctx, query, time.Now())
}

func (p PrometheusClient) VectorValue(ctx context.Context, query string) (prommodel.SampleValue, error) {
	result, _, err := p.Query(ctx, query)
	if err != nil {
		return 0, err
	}
	if result.Type() != prommodel.ValVector {
		return 0, fmt.Errorf("unexpected result type: %s", result.Type())
	}
	vec := result.(prommodel.Vector)
	if len(vec) == 0 {
		return 0, fmt.Errorf("no data returned")
	}
	return vec[0].Value, nil

}
