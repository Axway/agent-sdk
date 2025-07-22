package metric

import (
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/cmd"
	"github.com/Axway/agent-sdk/pkg/traceability/sampling"
	"github.com/Axway/agent-sdk/pkg/transaction/models"
	transutil "github.com/Axway/agent-sdk/pkg/transaction/util"
)

func centralMetricFromAPIMetric(in *APIMetric) *centralMetric {
	if in == nil {
		return nil
	}

	out := &centralMetric{
		EventID: in.EventID,
		Observation: &models.ObservationDetails{
			Start: in.Observation.Start,
		},
		Reporter: &Reporter{
			AgentVersion:     cmd.BuildVersion,
			AgentType:        cmd.BuildAgentName,
			AgentSDKVersion:  cmd.SDKBuildVersion,
			AgentName:        agent.GetCentralConfig().GetAgentName(),
			ObservationDelta: in.Observation.End - in.Observation.Start,
		},
	}

	if in.Unit == nil {
		status := in.Status
		if status == "" {
			status = sampling.GetStatusFromCodeString(in.StatusCode).String()
		}
		// transaction units
		out.Units = &Units{
			Transactions: &Transactions{
				UnitCount: UnitCount{
					Count: in.Count,
				},
				Status: status,
				Response: &ResponseMetrics{
					Max: in.Response.Max,
					Min: in.Response.Min,
					Avg: in.Response.Avg,
				},
			},
		}
		if in.Quota.ID != unknown && in.Quota.ID != "" {
			out.Units.Transactions.Quota = &models.ResourceReference{
				ID: in.Quota.ID,
			}
		}
	} else {
		// custom units
		out.Units = &Units{
			CustomUnits: map[string]*UnitCount{
				in.Unit.Name: {
					Count: in.Count,
				},
			},
		}
		if in.Quota.ID != unknown && in.Quota.ID != "" {
			out.Units.CustomUnits[in.Unit.Name].Quota = &models.ResourceReference{
				ID: in.Quota.ID,
			}
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

	return out
}

func splitMetricKey(key string) (string, string) {
	const delimiter = "."

	groupKey := strings.Join(strings.Split(key, delimiter)[:4], delimiter)
	metricKey := strings.Split(key, delimiter)[4]
	return groupKey, metricKey
}
