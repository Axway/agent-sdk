package metric

import (
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
)

func centralMetricFromAPIMetric(in *APIMetric) *centralMetric {
	out := &centralMetric{
		EventID: in.EventID,
		Observation: &ObservationDetails{
			Start: in.Observation.Start,
		},
	}

	if in.Unit == nil {
		// transaction units
		out.Units = &Units{
			Transactions: &Transactions{
				UnitCount: UnitCount{
					Count: in.Count,
				},
				Status: in.Status,
			},
		}
	} else {
		// custom units
		out.Units.CustomUnits[in.Unit.Name] = &UnitCount{
			Count: in.Count,
		}
	}

	if in.Subscription.ID != unknown && in.Subscription.ID != "" {
		out.Subscription = &models.ResourceReference{
			ID: in.Subscription.ID,
		}
	}

	if in.App.ID != unknown && in.App.ID != "" {
		out.App = &models.ApplicationResourceReference{
			ResourceReference: models.ResourceReference{
				ID: in.App.ID,
			},
			ConsumerOrgID: in.App.ConsumerOrgID,
		}
	}

	if in.Product.ID != unknown && in.Product.ID != "" {
		out.Product = &models.ProductResourceReference{
			ResourceReference: models.ResourceReference{
				ID: in.Product.ID,
			},
			VersionID: in.Product.VersionID,
		}
	}

	if in.API.ID != unknown && in.API.ID != "" {
		out.API = &models.APIResourceReference{
			ResourceReference: models.ResourceReference{
				ID: in.API.ID,
			},
			Name: in.API.Name,
		}
		svc := agent.GetCacheManager().GetAPIServiceWithAPIID(strings.TrimPrefix(in.API.ID, transutil.SummaryEventProxyIDPrefix))
		if svc != nil {
			out.API.APIServiceID = svc.Metadata.ID
		}
	}

	if in.AssetResource.ID != unknown && in.AssetResource.ID != "" {
		out.AssetResource = &models.ResourceReference{
			ID: in.AssetResource.ID,
		}
	}

	if in.ProductPlan.ID != unknown && in.ProductPlan.ID != "" {
		out.ProductPlan = &models.ResourceReference{
			ID: in.ProductPlan.ID,
		}
	}

	if in.Quota.ID != unknown && in.Quota.ID != "" {
		out.Units.Transactions.Quota = &models.ResourceReference{
			ID: in.Quota.ID,
		}
	}

	return out
}

func splitMetricKey(key string) (string, string) {
	const delimiter = "."

	groupKey := strings.Join(strings.Split(key, delimiter)[:4], delimiter)
	metricKey := strings.Split(key, delimiter)[4]
	return groupKey, metricKey
}
