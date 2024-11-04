package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Axway/agent-sdk/pkg/agent/cache"
	"github.com/Axway/agent-sdk/pkg/amplify/agent/customunits"
	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/catalog/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/customunit"
	"github.com/Axway/agent-sdk/pkg/util"
)

type customUnitQuotaHandler struct {
	configs []config.MetricServiceConfiguration
	cache   cache.Manager
}

type CustomUnitQuotaHandler interface {
	Handle(context.Context, *management.AccessRequest, *management.ManagedApplication) (string, error)
}

// NewCustomUnitQuotaHandler creates a new Custom Unit Quota Handler for AccessRequest processing
func NewCustomUnitQuotaHandler(configs []config.MetricServiceConfiguration) CustomUnitQuotaHandler {
	return &customUnitQuotaHandler{
		configs: configs,
	}
}

func (h *customUnitQuotaHandler) Handle(ctx context.Context, ar *management.AccessRequest, app *management.ManagedApplication) (string, error) {
	// Build quota info
	quotaInfo, err := h.buildQuotaInfo(ctx, ar, app)
	if err != nil {
		return "", fmt.Errorf("could not build quota info from access request")
	}
	errMessage := ""
	for _, config := range h.configs {
		if config.MetricServiceEnabled() {
			factory := customunit.NewQuotaEnforcementClientFactory(config.URL, quotaInfo)
			client, _ := factory(ctx)
			response, err := client.QuotaEnforcementInfo()
			if err != nil {
				// if error from QE and reject on fail, we return the error back to the central
				if response.Error != "" && config.RejectOnFailEnabled() {
					errMessage = errMessage + fmt.Sprintf("TODO: message: %s", err.Error())
				}
			}
		}
	}

	return errMessage, nil
}

func (h *customUnitQuotaHandler) buildQuotaInfo(ctx context.Context, ar *management.AccessRequest, app *management.ManagedApplication) (*customunits.QuotaInfo, error) {
	unitRef, count := h.getQuotaInfo(ar)
	if unitRef == "" {
		return nil, nil
	}

	instance, err := h.getServiceInstance(ctx, ar)
	if err != nil {
		return nil, err
	}

	// Get service instance from access request to fetch the api service
	serviceRef := instance.GetReferenceByGVK(management.APIServiceGVK())
	service := h.cache.GetAPIServiceWithName(serviceRef.Name)
	if service == nil {
		return nil, fmt.Errorf("could not find service connected to quota")
	}
	extAPIID, err := util.GetAgentDetailsValue(instance, defs.AttrExternalAPIID)
	if err != nil {
		return nil, err
	}

	q := &customunits.QuotaInfo{
		ApiInfo: &customunits.APIInfo{
			ServiceDetails: util.GetAgentDetailStrings(service),
			ServiceName:    service.Name,
			ServiceID:      service.Metadata.ID,
			ExternalAPIID:  extAPIID,
		},
		AppInfo: &customunits.AppInfo{
			AppDetails: util.GetAgentDetailStrings(app),
			AppName:    app.Name,
			AppID:      app.Metadata.ID,
		},
		Quota: &customunits.Quota{
			Count: int64(count),
			Unit:  unitRef,
		},
	}

	return q, nil
}

type reference struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	Unit string `json:"unit"`
}

func (h *customUnitQuotaHandler) getQuotaInfo(ar *management.AccessRequest) (string, int) {
	index := 0
	if len(ar.Spec.AdditionalQuotas) < index+1 {
		return "", 0
	}

	q := ar.Spec.AdditionalQuotas[index]
	for _, r := range ar.References {
		d, _ := json.Marshal(r)
		ref := &reference{}
		json.Unmarshal(d, ref)
		if ref.Kind == catalog.QuotaGVK().Kind && ref.Name == q.Name {
			return ref.Unit, int(q.Limit)
		}
	}
	return "", 0
}

func (h *customUnitQuotaHandler) getServiceInstance(_ context.Context, ar *management.AccessRequest) (*apiv1.ResourceInstance, error) {
	instRef := ar.GetReferenceByGVK(management.APIServiceInstanceGVK())
	instID := instRef.ID
	instance, err := h.cache.GetAPIServiceInstanceByID(instID)
	if err != nil {
		return nil, err
	}
	return instance, nil
}
