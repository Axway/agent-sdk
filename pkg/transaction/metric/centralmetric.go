package metric

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Axway/agent-sdk/pkg/transaction/models"
	"github.com/rcrowley/go-metrics"
	"github.com/sirupsen/logrus"
)

type groupedMetrics struct {
	lock       *sync.Mutex
	counters   map[string]metrics.Counter
	histograms map[string]metrics.Histogram
}

func newGroupedMetric() groupedMetrics {
	return groupedMetrics{
		lock:       &sync.Mutex{},
		counters:   make(map[string]metrics.Counter),
		histograms: make(map[string]metrics.Histogram),
	}
}

func (g groupedMetrics) getOrCreateHistogram(key string) metrics.Histogram {
	g.lock.Lock()
	defer g.lock.Unlock()

	if _, ok := g.histograms[key]; !ok {
		sampler := metrics.NewUniformSample(2048)
		g.histograms[key] = metrics.NewHistogram(sampler)
	}
	return g.histograms[key]
}

func (g groupedMetrics) getOrCreateCounter(key string) metrics.Counter {
	g.lock.Lock()
	defer g.lock.Unlock()

	if _, ok := g.counters[key]; !ok {
		g.counters[key] = metrics.NewCounter()
	}
	return g.counters[key]
}

type CentralMetricBuilder struct {
	*centralMetric
}

func NewCentralMetricBuilder() *CentralMetricBuilder {
	return &CentralMetricBuilder{
		centralMetric: &centralMetric{},
	}
}

func (b *CentralMetricBuilder) SetSubscription(sub *models.ResourceReference) *CentralMetricBuilder {
	b.Subscription = sub
	return b
}

func (b *CentralMetricBuilder) SetApp(app *models.ApplicationResourceReference) *CentralMetricBuilder {
	b.App = app
	return b
}

func (b *CentralMetricBuilder) SetProduct(product *models.ProductResourceReference) *CentralMetricBuilder {
	b.Product = product
	return b
}

func (b *CentralMetricBuilder) SetAPI(api *models.APIResourceReference) *CentralMetricBuilder {
	b.API = api
	return b
}

func (b *CentralMetricBuilder) SetAssetResource(asset *models.ResourceReference) *CentralMetricBuilder {
	b.AssetResource = asset
	return b
}

func (b *CentralMetricBuilder) SetProductPlan(plan *models.ResourceReference) *CentralMetricBuilder {
	b.ProductPlan = plan
	return b
}

func (b *CentralMetricBuilder) SetUnits(units *Units) *CentralMetricBuilder {
	b.Units = units
	return b
}

func (b *CentralMetricBuilder) SetQuota(quota *models.ResourceReference) *CentralMetricBuilder {
	if b.Units != nil && b.Units.Transactions != nil {
		b.Units.Transactions.Quota = quota
	} else if b.Units == nil {
		b.Units = &Units{
			Transactions: &Transactions{
				UnitCount: UnitCount{
					Quota: quota,
				},
			},
		}
	} else if b.Units.Transactions == nil {
		b.Units.Transactions = &Transactions{
			UnitCount: UnitCount{
				Quota: quota,
			},
		}
	}
	return b
}

func (b *CentralMetricBuilder) SetObservation(obs *models.ObservationDetails) *CentralMetricBuilder {
	b.Observation = obs
	return b
}

func (b *CentralMetricBuilder) SetEventID(id string) *CentralMetricBuilder {
	b.EventID = id
	return b
}

func (b *CentralMetricBuilder) Build() *centralMetric {
	return b.centralMetric
}

type centralMetric struct {
	Subscription  *models.ResourceReference            `json:"subscription,omitempty"`
	App           *models.ApplicationResourceReference `json:"application,omitempty"`
	Product       *models.ProductResourceReference     `json:"product,omitempty"`
	API           *models.APIResourceReference         `json:"api,omitempty"`
	AssetResource *models.ResourceReference            `json:"assetResource,omitempty"`
	ProductPlan   *models.ResourceReference            `json:"productPlan,omitempty"`
	Units         *Units                               `json:"units,omitempty"`
	Reporter      *Reporter                            `json:"reporter,omitempty"`
	Observation   *models.ObservationDetails           `json:"-"`
	EventID       string                               `json:"-"`
}

// GetStartTime - Returns the start time for subscription metric
func (a *centralMetric) GetStartTime() time.Time {
	return time.UnixMilli(a.Observation.Start)
}

// GetType - Returns APIMetric
func (a *centralMetric) GetType() string {
	return "APIMetric"
}

// GetType - Returns APIMetric
func (a *centralMetric) GetEventID() string {
	return a.EventID
}

func (a *centralMetric) GetLogFields() logrus.Fields {
	fields := logrus.Fields{
		"id":             a.EventID,
		"startTimestamp": a.Observation.Start,
		"endTimestamp":   a.Observation.End,
	}
	if a.Subscription != nil {
		fields = a.Subscription.GetLogFields(fields, "subscriptionID")
	}
	if a.App != nil {
		fields = a.App.GetLogFields(fields, "applicationID")
	}
	if a.Product != nil {
		fields = a.Product.GetLogFields(fields, "productID")
	}
	if a.API != nil {
		fields = a.API.GetLogFields(fields, "apiID")
	}
	if a.AssetResource != nil {
		fields = a.AssetResource.GetLogFields(fields, "assetResourceID")
	}
	if a.ProductPlan != nil {
		fields = a.ProductPlan.GetLogFields(fields, "productPlanID")
	}

	// add transaction unit info and custom units if they exist
	if a.Units == nil {
		return fields
	}
	if a.Units.Transactions != nil {
		if a.Units.Transactions.Quota != nil {
			fields = a.Units.Transactions.Quota.GetLogFields(fields, "transactionQuotaID")
		}
		fields["transactionCount"] = a.Units.Transactions.Count
		fields["status"] = a.Units.Transactions.Status
		fields["minResponse"] = a.Units.Transactions.Response.Min
		fields["maxResponse"] = a.Units.Transactions.Response.Max
		fields["avgResponse"] = a.Units.Transactions.Response.Avg
	}
	if len(a.Units.CustomUnits) == 0 {
		return fields
	}
	for k, u := range a.Units.CustomUnits {
		if u.Quota != nil {
			fields = u.Quota.GetLogFields(fields, fmt.Sprintf("%sQuotaID", k))
		}
		fields[fmt.Sprintf("%sCount", k)] = u.Count
	}
	return fields
}

// getKey - returns the cache key for the metric
func (a *centralMetric) getKey() string {
	subID := unknown
	if a.Subscription != nil {
		subID = a.Subscription.ID
	}
	appID := unknown
	if a.App != nil {
		appID = a.App.ID
	}
	apiID := unknown
	if a.API != nil {
		apiID = a.API.ID
	}
	uniqueKey := unknown
	if a.Units != nil && a.Units.Transactions != nil && a.Units.Transactions.Status != "" {
		uniqueKey = a.Units.Transactions.Status
	} else {
		// get the first, and should be only, custom unit name
		for k := range a.Units.CustomUnits {
			uniqueKey = k
			break
		}
	}

	return strings.Join([]string{metricKeyPrefix, subID, appID, apiID, uniqueKey}, ".")
}

// getKey - returns the cache key for the metric
func (a *centralMetric) getKeyParts() (string, string, string, string) {
	key := a.getKey()
	parts := strings.Split(key, ".")
	return parts[1], parts[2], parts[3], parts[4]
}

func (a *centralMetric) createCachedMetric(cached cachedMetricInterface) cachedMetric {
	cacheM := cachedMetric{
		Subscription:  a.Subscription,
		App:           a.App,
		Product:       a.Product,
		API:           a.API,
		AssetResource: a.AssetResource,
		ProductPlan:   a.ProductPlan,
		Count:         cached.Count(),
		Values:        cached.Values(),
	}

	if a.Units.Transactions != nil {
		cacheM.Quota = a.Units.Transactions.Quota
		cacheM.StatusCode = a.Units.Transactions.Status
	} else {
		for u := range a.Units.CustomUnits {
			cacheM.Unit = &models.Unit{
				Name: u,
			}
		}
	}
	return cacheM
}
