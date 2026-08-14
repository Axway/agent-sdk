package transaction

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/metric"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/stretchr/testify/assert"
)

func TestEventReportBuilderLLMUnits(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Second)

	report, err := NewEventReportBuilder().
		SetProxy(Proxy{ID: "api-1", Name: "api-name", Stage: "prod"}).
		SetApplication(Application{ID: "app-1", Name: "app-name"}).
		SetLLMModel("gpt-4").
		SetUnitAmount(metric.LLMRequests, 1, start, end).
		SetUnitAmount(metric.LLMInputTokens, 120, start, end).
		SetUnitAmount(metric.LLMOutputTokens, 45, start, end).
		SetOnlyTrackMetrics(true).
		Build()

	assert.NoError(t, err)
	batch := report.GetMetricsBatch()
	assert.Len(t, batch, 1, "all llm units should collapse into a single metric")

	llm, ok := batch[0].(metric.LLMMetricDetail)
	assert.True(t, ok, "expected an LLMMetricDetail")
	assert.Equal(t, "gpt-4", llm.Model)
	assert.Equal(t, "api-1", llm.APIDetails.ID)
	assert.Equal(t, "app-1", llm.AppDetails.ID)
	assert.Equal(t, int64(1), llm.Units[metric.LLMRequests])
	assert.Equal(t, int64(120), llm.Units[metric.LLMInputTokens])
	assert.Equal(t, int64(45), llm.Units[metric.LLMOutputTokens])
	assert.Equal(t, start.UnixMilli(), llm.Observation.Start)
	assert.Equal(t, end.UnixMilli(), llm.Observation.End)
}

func TestEventReportBuilderCustomUnits(t *testing.T) {
	now := time.Now()

	report, err := NewEventReportBuilder().
		SetProxy(Proxy{ID: "api-1", Name: "api-name"}).
		SetApplication(Application{ID: "app-1", Name: "app-name"}).
		SetUnitAmount(metric.CostUSDUnit, 10, now, now).
		SetUnitAmount(metric.LLMTotalTokens, 200, now, now).
		SetOnlyTrackMetrics(true).
		Build()

	assert.NoError(t, err)
	batch := report.GetMetricsBatch()
	assert.Len(t, batch, 2, "without a model each unit is reported separately")
	for _, item := range batch {
		_, ok := item.(models.CustomMetricDetail)
		assert.True(t, ok, "expected a CustomMetricDetail")
	}
}

func TestEventReportBuilderUnitsRequireProxyAndApp(t *testing.T) {
	_, err := NewEventReportBuilder().
		SetLLMModel("gpt-4").
		SetUnitAmount(metric.LLMRequests, 1, time.Now(), time.Now()).
		SetOnlyTrackMetrics(true).
		Build()

	assert.Error(t, err)
}

// reportWithUnits returns an eventReport wired with a proxy/app and the given units, ready for
// exercising the metric-unit handling methods directly.
func reportWithUnits(model string, units map[metric.UnitType]unitDetails) *eventReport {
	return &eventReport{
		proxy:        &Proxy{ID: "api-1", Name: "api-name", Stage: "prod"},
		app:          &Application{ID: "app-1", Name: "app-name"},
		llmModel:     model,
		units:        units,
		metricsBatch: []interface{}{},
	}
}

func TestHandleUnits(t *testing.T) {
	unit := unitDetails{count: 1, startTime: time.Now(), endTime: time.Now()}

	t.Run("no units is a no-op", func(t *testing.T) {
		e := reportWithUnits("", map[metric.UnitType]unitDetails{})
		assert.NoError(t, e.handleUnits())
		assert.Empty(t, e.metricsBatch)
	})

	t.Run("requires proxy and app when units are present", func(t *testing.T) {
		e := reportWithUnits("", map[metric.UnitType]unitDetails{metric.LLMRequests: unit})
		e.proxy = nil
		assert.Error(t, e.handleUnits())

		e = reportWithUnits("", map[metric.UnitType]unitDetails{metric.LLMRequests: unit})
		e.app = nil
		assert.Error(t, e.handleUnits())
	})

	t.Run("dispatches to a single llm metric when a model is set", func(t *testing.T) {
		e := reportWithUnits("gpt-4", map[metric.UnitType]unitDetails{
			metric.LLMRequests:    unit,
			metric.LLMInputTokens: unit,
		})
		assert.NoError(t, e.handleUnits())
		assert.Len(t, e.metricsBatch, 1)
		_, ok := e.metricsBatch[0].(metric.LLMMetricDetail)
		assert.True(t, ok)
	})

	t.Run("dispatches to per-unit custom metrics without a model", func(t *testing.T) {
		e := reportWithUnits("", map[metric.UnitType]unitDetails{
			metric.CostUSDUnit:    unit,
			metric.LLMInputTokens: unit,
		})
		assert.NoError(t, e.handleUnits())
		assert.Len(t, e.metricsBatch, 2)
		for _, item := range e.metricsBatch {
			_, ok := item.(models.CustomMetricDetail)
			assert.True(t, ok)
		}
	})
}

func TestHandleLLMMetric(t *testing.T) {
	t0 := time.Now()
	t1 := t0.Add(1 * time.Second)
	t2 := t0.Add(2 * time.Second)
	t3 := t0.Add(3 * time.Second)

	e := reportWithUnits("gpt-4", map[metric.UnitType]unitDetails{
		metric.LLMInputTokens:  {count: 120, startTime: t0, endTime: t2},
		metric.LLMOutputTokens: {count: 45, startTime: t1, endTime: t3},
	})

	api := models.APIDetails{ID: "api-1", Name: "api-name", Stage: "prod"}
	app := models.AppDetails{ID: "app-1", Name: "app-name"}
	e.handleLLMMetric(api, app)

	assert.Len(t, e.metricsBatch, 1)
	llm := e.metricsBatch[0].(metric.LLMMetricDetail)

	assert.Equal(t, "gpt-4", llm.Model)
	assert.Equal(t, api, llm.APIDetails)
	assert.Equal(t, app, llm.AppDetails)
	assert.Equal(t, int64(120), llm.Units[metric.LLMInputTokens])
	assert.Equal(t, int64(45), llm.Units[metric.LLMOutputTokens])
	// observation spans the earliest start to the latest end across all units
	assert.Equal(t, t0.UnixMilli(), llm.Observation.Start)
	assert.Equal(t, t3.UnixMilli(), llm.Observation.End)
}

func TestAppendCustomMetrics(t *testing.T) {
	start := time.Now()
	end := start.Add(time.Second)

	e := reportWithUnits("", map[metric.UnitType]unitDetails{
		metric.CostUSDUnit: {count: 10, startTime: start, endTime: end},
	})

	api := models.APIDetails{ID: "api-1", Name: "api-name", Stage: "prod"}
	app := models.AppDetails{ID: "app-1", Name: "app-name"}
	e.appendCustomMetrics(api, app)

	assert.Len(t, e.metricsBatch, 1)
	custom := e.metricsBatch[0].(models.CustomMetricDetail)

	assert.Equal(t, api, custom.APIDetails)
	assert.Equal(t, app, custom.AppDetails)
	assert.Equal(t, metric.CostUSDUnit.String(), custom.UnitDetails.Name)
	assert.Equal(t, int64(10), custom.Count)
	assert.Equal(t, start.UnixMilli(), custom.Observation.Start)
	assert.Equal(t, end.UnixMilli(), custom.Observation.End)
}
