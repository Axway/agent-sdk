package metric

import (
	"github.com/Axway/agent-sdk/pkg/transaction/models"
)

func centralMetricFromAPIMetric(in *APIMetric) *centralMetricEvent {
	out := &centralMetricEvent{
		StatusCode: in.StatusCode,
		Status:     in.Status,
		Count:      in.Count,
		EventID:    in.EventID,
		StartTime:  in.StartTime,
	}

	if in.Subscription.ID != unknown && in.Subscription.ID != "" {
		out.Subscription = &models.Subscription{
			ID:   in.Subscription.ID,
			Name: in.Subscription.Name,
		}
	}

	if in.App.ID != unknown && in.App.ID != "" {
		out.App = &models.AppDetails{
			ID:            in.App.ID,
			Name:          in.App.Name,
			ConsumerOrgID: in.App.ConsumerOrgID,
		}
	}

	if in.Product.ID != unknown && in.Product.ID != "" {
		out.Product = &models.Product{
			ID:          in.Product.ID,
			Name:        in.Product.Name,
			VersionName: in.Product.VersionName,
			VersionID:   in.Product.VersionID,
		}
	}

	if in.API.ID != unknown && in.API.ID != "" {
		out.API = &models.APIDetails{
			ID:                 in.API.ID,
			Name:               in.API.Name,
			Revision:           in.API.Revision,
			TeamID:             in.API.TeamID,
			APIServiceInstance: in.API.APIServiceInstance,
			Stage:              in.API.Stage,
			Version:            in.API.Version,
		}
	}

	if in.AssetResource.ID != unknown && in.AssetResource.ID != "" {
		out.AssetResource = &models.AssetResource{
			ID:   in.AssetResource.ID,
			Name: in.AssetResource.Name,
		}
	}

	if in.ProductPlan.ID != unknown && in.ProductPlan.ID != "" {
		out.ProductPlan = &models.ProductPlan{
			ID: in.ProductPlan.ID,
		}
	}

	if in.Quota.ID != unknown && in.Quota.ID != "" {
		out.Quota = &models.Quota{
			ID: in.Quota.ID,
		}
	}

	if in.Unit.ID != unknown && in.Unit.ID != "" {
		out.Unit = &models.Unit{
			ID:   in.Unit.ID,
			Name: in.Unit.Name,
		}
	}

	return out
}
