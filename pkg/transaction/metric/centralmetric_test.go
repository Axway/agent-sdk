package metric

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func TestCentralMetricBuilderSetVersion(t *testing.T) {
	cases := map[string]struct {
		version string
	}{
		"sets version 3":    {version: "3"},
		"sets empty string": {version: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetVersion(tc.version)
			assert.Equal(t, tc.version, b.Build().Version)
		})
	}
}

func TestCentralMetricBuilderSetAPICDeployment(t *testing.T) {
	cases := map[string]struct {
		deployment string
	}{
		"sets prod":  {deployment: "prod"},
		"sets teams": {deployment: "teams"},
		"sets empty": {deployment: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetAPICDeployment(tc.deployment)
			assert.Equal(t, tc.deployment, b.Build().APICDeployment)
		})
	}
}

func TestCentralMetricBuilderSetEnvironmentRuntimeType(t *testing.T) {
	cases := map[string]struct {
		calls []string
		want  string
	}{
		"sets managed":              {calls: []string{"managed"}, want: "managed"},
		"sets unmanaged":            {calls: []string{"unmanaged"}, want: "unmanaged"},
		"sets unknown":              {calls: []string{"unknown"}, want: "unknown"},
		"overwrites on second call": {calls: []string{"managed", "unmanaged"}, want: "unmanaged"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder()
			for _, rt := range tc.calls {
				b.SetEnvironmentRuntimeType(rt)
			}
			result := b.Build()
			assert.NotNil(t, result.Environment)
			assert.Equal(t, tc.want, result.Environment.RuntimeType)
		})
	}
}

func TestCentralMetricBuilderSetAPIServiceRevision(t *testing.T) {
	ref := &models.ResourceReference{ID: "rev-123"}
	cases := map[string]struct {
		ref  *models.ResourceReference
		want *models.ResourceReference
	}{
		"sets reference": {ref: ref, want: ref},
		"sets nil":       {ref: nil, want: nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetAPIServiceRevision(tc.ref)
			assert.Equal(t, tc.want, b.Build().APIServiceRevision)
		})
	}
}

func TestCentralMetricBuilderSetReporter(t *testing.T) {
	reporter := &Reporter{AgentName: "test-agent", AgentVersion: "1.0.0"}
	cases := map[string]struct {
		reporter *Reporter
		want     *Reporter
	}{
		"sets reporter": {reporter: reporter, want: reporter},
		"sets nil":      {reporter: nil, want: nil},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := NewCentralMetricBuilder().SetReporter(tc.reporter)
			assert.Equal(t, tc.want, b.Build().Reporter)
		})
	}
}

func refs(sub, app, api string) (*models.ResourceReference, *models.ApplicationResourceReference, *models.APIResourceReference) {
	return &models.ResourceReference{ID: sub},
		&models.ApplicationResourceReference{ResourceReference: models.ResourceReference{ID: app}},
		&models.APIResourceReference{ResourceReference: models.ResourceReference{ID: api}}
}

func TestCentralMetricGetKey(t *testing.T) {
	sub, app, api := refs("sub", "app", "api")

	cases := map[string]struct {
		metric *centralMetric
		want   string
	}{
		"transaction status keys on status": {
			metric: &centralMetric{
				Subscription: sub, App: app, API: api,
				Units: &Units{Transactions: &Transactions{Status: "Success"}},
			},
			want: "metric.sub.app.api.Success",
		},
		"custom unit keys on unit name": {
			metric: &centralMetric{
				Subscription: sub, App: app, API: api,
				Units: &Units{Units: map[string]*UnitCount{"cost-usd": {}}},
			},
			want: "metric.sub.app.api.cost-usd",
		},
		"llm keys on model regardless of units": {
			metric: &centralMetric{
				Subscription: sub, App: app, API: api,
				LLM:   &models.LLMReference{Model: "gpt-4"},
				Units: &Units{Units: map[string]*UnitCount{"llm-requests": {}}},
			},
			want: "metric.sub.app.api.gpt-4",
		},
		"missing references fall back to unknown": {
			metric: &centralMetric{
				Units: &Units{Transactions: &Transactions{Status: "Success"}},
			},
			want: "metric.unknown.unknown.unknown.Success",
		},
		"nil units fall back to unknown leaf": {
			metric: &centralMetric{Subscription: sub, App: app, API: api},
			want:   "metric.sub.app.api.unknown",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.metric.getKey())

			s, a, i, leaf := tc.metric.getKeyParts()
			parts := []string{"metric", s, a, i, leaf}
			assert.Equal(t, tc.want, join(parts))
		})
	}
}

func join(parts []string) string {
	out := parts[0]
	for _, p := range parts[1:] {
		out += "." + p
	}
	return out
}

func TestCentralMetricGetLLMInfo(t *testing.T) {
	m := &centralMetric{LLM: &models.LLMReference{ResourceReference: models.ResourceReference{ID: "llm-1"}, Model: "gpt-4"}}
	id, model := m.GetLLMInfo()
	assert.Equal(t, "llm-1", id)
	assert.Equal(t, "gpt-4", model)

	empty := &centralMetric{}
	id, model = empty.GetLLMInfo()
	assert.Empty(t, id)
	assert.Empty(t, model)
}

type fakeCachedMetric struct {
	count  int64
	values []int64
}

func (f fakeCachedMetric) Count() int64    { return f.count }
func (f fakeCachedMetric) Values() []int64 { return f.values }

func TestCentralMetricCreateCachedMetric(t *testing.T) {
	sub, app, api := refs("sub", "app", "api")

	t.Run("transaction preserves quota and status", func(t *testing.T) {
		m := &centralMetric{
			Subscription: sub, App: app, API: api,
			Units: &Units{Transactions: &Transactions{
				UnitCount: UnitCount{Quota: &models.ResourceReference{ID: "quota-1"}},
				Status:    "Success",
			}},
		}
		cached := m.createCachedMetric(fakeCachedMetric{count: 7, values: []int64{1, 2}})

		assert.Equal(t, int64(7), cached.Count)
		assert.Equal(t, []int64{1, 2}, cached.Values)
		assert.Equal(t, "Success", cached.StatusCode)
		assert.NotNil(t, cached.Quota)
		assert.Equal(t, "quota-1", cached.Quota.ID)
		assert.Nil(t, cached.Unit)
	})

	t.Run("llm preserves llm reference and unit name", func(t *testing.T) {
		m := &centralMetric{
			Subscription: sub, App: app, API: api,
			LLM:   &models.LLMReference{Model: "gpt-4"},
			Units: &Units{Units: map[string]*UnitCount{"llm-inputtokens": {}}},
		}
		cached := m.createCachedMetric(fakeCachedMetric{count: 120})

		assert.Equal(t, int64(120), cached.Count)
		assert.NotNil(t, cached.LLM)
		assert.Equal(t, "gpt-4", cached.LLM.Model)
		assert.NotNil(t, cached.Unit)
		assert.Equal(t, "llm-inputtokens", cached.Unit.Name)
		assert.Empty(t, cached.StatusCode)
	})
}
