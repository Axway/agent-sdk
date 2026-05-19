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

	apicDeployment, agentName, runtimeType := centralConfigFields()

	out := &centralMetric{
		Version:        metricDataVersion,
		APICDeployment: apicDeployment,
		Environment:    &EnvironmentInfo{RuntimeType: runtimeType},
		EventID:        in.EventID,
		Observation:    &models.ObservationDetails{Start: in.Observation.Start, End: in.Observation.End},
		Reporter: &Reporter{
			AgentVersion:     cmd.BuildVersion,
			AgentType:        cmd.BuildAgentName,
			AgentSDKVersion:  cmd.SDKBuildVersion,
			AgentName:        agentName,
			ObservationDelta: in.Observation.End - in.Observation.Start,
		},
	}

	out.Units = buildUnits(in)

	if id := in.Subscription.ID; isKnownID(id) {
		out.Subscription = &models.ResourceReference{ID: id}
	}

	if isKnownID(in.App.ID) {
		out.App = buildAppRef(in.App)
	}

	if id := in.Product.ID; isKnownID(id) {
		out.Product = &models.ProductResourceReference{
			ResourceReference: models.ResourceReference{ID: id},
			VersionID:         in.Product.VersionID,
		}
	}

	if isKnownID(in.API.ID) {
		out.API = buildAPIRef(in.API)
	}

	if id := in.AssetResource.ID; isKnownID(id) {
		out.AssetResource = &models.ResourceReference{ID: id}
	}

	if id := in.ProductPlan.ID; isKnownID(id) {
		out.ProductPlan = &models.ResourceReference{ID: id}
	}

	return out
}

func centralConfigFields() (apicDeployment, agentName, runtimeType string) {
	runtimeType = unknown
	cfg := agent.GetCentralConfig()
	if cfg == nil {
		return
	}
	if cfg.IsAxwayManaged() {
		runtimeType = runtimeTypeManaged
	} else {
		runtimeType = runtimeTypeUnmanaged
	}
	apicDeployment = cfg.GetAPICDeployment()
	agentName = cfg.GetAgentName()
	return
}

func isKnownID(id string) bool {
	return id != "" && id != unknown
}

func buildUnits(in *APIMetric) *Units {
	if in.Unit != nil {
		return buildCustomUnits(in)
	}
	return buildTransactionUnits(in)
}

func buildTransactionUnits(in *APIMetric) *Units {
	status := in.Status
	if status == "" {
		status = sampling.GetStatusFromCodeString(in.StatusCode).String()
	}
	txn := &Transactions{
		UnitCount: UnitCount{Count: in.Count},
		Duration:  in.Observation.End - in.Observation.Start,
		Status:    status,
		Response: &ResponseMetrics{
			Max: in.Response.Max,
			Min: in.Response.Min,
			Avg: in.Response.Avg,
		},
	}
	if isKnownID(in.Quota.ID) {
		txn.Quota = &models.ResourceReference{ID: in.Quota.ID}
	}
	return &Units{Transactions: txn}
}

func buildCustomUnits(in *APIMetric) *Units {
	uc := &UnitCount{Count: in.Count}
	if isKnownID(in.Quota.ID) {
		uc.Quota = &models.ResourceReference{ID: in.Quota.ID}
	}
	return &Units{
		CustomUnits: map[string]*UnitCount{in.Unit.Name: uc},
	}
}

func buildAppRef(app models.AppDetails) *models.ApplicationResourceReference {
	ref := &models.ApplicationResourceReference{
		ResourceReference: models.ResourceReference{ID: app.ID},
		ConsumerOrgID:     app.ConsumerOrgID,
	}
	ref.Owner = resolveAppOwnerFromCache(app.ID)
	return ref
}

func resolveAppOwnerFromCache(appID string) *models.OwnerBlock {
	cacheManager := agent.GetCacheManager()
	if cacheManager == nil {
		return &models.OwnerBlock{Type: unknown}
	}
	managedApp := cacheManager.GetManagedApplicationByApplicationID(appID)
	if managedApp == nil {
		managedApp = cacheManager.GetManagedApplication(appID)
	}
	if managedApp != nil {
		return transutil.ResolveAppOwnerFromManagedApp(managedApp)
	}
	return &models.OwnerBlock{Type: unknown}
}

func buildAPIRef(api models.APIDetails) *models.APIResourceReference {
	ref := &models.APIResourceReference{
		ResourceReference: models.ResourceReference{ID: api.ID},
		Name:              api.Name,
	}
	cacheManager := agent.GetCacheManager()
	stripped := strings.TrimPrefix(api.ID, transutil.SummaryEventProxyIDPrefix)
	if svc := cacheManager.GetAPIServiceWithAPIID(stripped); svc != nil {
		ref.APIServiceID = svc.Metadata.ID
	}
	ref.Owner = transutil.ResolveAPIOwner(api.ID, cacheManager)
	return ref
}

func splitMetricKey(key string) (string, string) {
	const delimiter = "."

	groupKey := strings.Join(strings.Split(key, delimiter)[:4], delimiter)
	metricKey := strings.Split(key, delimiter)[4]
	return groupKey, metricKey
}
