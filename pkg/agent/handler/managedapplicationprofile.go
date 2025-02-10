package handler

import (
	"context"

	apiv1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	defs "github.com/Axway/agent-sdk/pkg/apic/definitions"
	prov "github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/util"
	"github.com/Axway/agent-sdk/pkg/util/log"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

type getTeamByID interface {
	GetTeamByID(id string) *defs.PlatformTeam
}

type managedApplicationProfileCache interface {
	getTeamByID
	GetApplicationProfileDefinitionByName(name string) (*v1.ResourceInstance, error)
}

type managedApplicationProfile struct {
	marketplaceHandler
	logger log.FieldLogger
	prov   prov.ApplicationProfileProvisioner
	cache  managedApplicationProfileCache
	client client
}

// NewManagedApplicationProfileHandler creates a Handler for Credentials
func NewManagedApplicationProfileHandler(prov prov.ApplicationProfileProvisioner, cache managedApplicationProfileCache, client client) Handler {
	return &managedApplicationProfile{
		logger: log.NewFieldLogger().WithComponent("managedApplicationProfile").WithPackage("agent.handler"),
		prov:   prov,
		cache:  cache,
		client: client,
	}
}

// Handle processes grpc events triggered for ManagedApplicationProfiles
func (h *managedApplicationProfile) Handle(ctx context.Context, meta *proto.EventMeta, resource *apiv1.ResourceInstance) error {
	action := GetActionFromContext(ctx)
	if resource.Kind != management.ManagedApplicationProfileGVK().Kind || h.prov == nil || h.shouldIgnoreSubResourceUpdate(action, meta) {
		return nil
	}

	log := getLoggerFromContext(ctx).WithComponent("managedApplicationProfileHandler")
	ctx = setLoggerInContext(ctx, log)

	profile := &management.ManagedApplicationProfile{}
	err := profile.FromInstance(resource)
	if err != nil {
		log.WithError(err).Error("could not handle application request")
		return nil
	}

	if ok := isStatusFound(profile.Status); !ok {
		log.Debug("could not handle application request as it did not have a status subresource")
		return nil
	}

	if ok := h.shouldProcessPending(profile.Status, profile.Metadata.State); ok {
		log.Trace("processing resource in pending status")
		return h.onPending(ctx, profile)
	}

	return nil
}

func (h *managedApplicationProfile) onPending(ctx context.Context, profile *management.ManagedApplicationProfile) error {
	log := getLoggerFromContext(ctx)

	defer func() {
		statusErr := h.client.CreateSubResource(profile.ResourceMeta, map[string]interface{}{"status": profile.Status})
		if statusErr != nil {
			log.WithError(statusErr).Error("error creating status subresources")
		}
	}()

	app, err := h.getManagedApp(ctx, profile)
	if err != nil {
		log.WithError(err).Error("error getting managed app")
		h.onError(ctx, profile, err)
		return err
	}

	h.checkForEnumValueMap(ctx, profile.Spec.Data, profile.Spec.ApplicationProfileDefinition)

	pma := provManagedAppProfile{
		attributes:        profile.Spec.Data,
		profileDefinition: profile.Spec.ApplicationProfileDefinition,
		managedAppName:    app.Name,
		teamName:          getTeamName(h.cache, app.Owner),
		data:              util.GetAgentDetails(app),
		consumerOrgID:     getConsumerOrgID(app),
		id:                app.Metadata.ID,
	}

	status := h.prov.ApplicationProfileRequestProvision(pma)

	profile.Status = prov.NewStatusReason(status)

	details := util.MergeMapStringString(util.GetAgentDetailStrings(profile), status.GetProperties())
	util.SetAgentDetails(profile, util.MapStringStringToMapStringInterface(details))

	profile.SubResources = map[string]interface{}{
		defs.XAgentDetails: util.GetAgentDetails(profile),
	}

	err = h.client.CreateSubResource(profile.ResourceMeta, profile.SubResources)
	if err != nil {
		log.WithError(err).Error("error creating subresources")
	}

	return err
}

func (h *managedApplicationProfile) checkForEnumValueMap(_ context.Context, data map[string]interface{}, profileDef string) {
	log := h.logger.WithField("applicationProfileDefinition", profileDef)

	// get application profile definition
	ri, err := h.cache.GetApplicationProfileDefinitionByName(profileDef)
	if err != nil {
		log.WithError(err).Error("error getting application profile definition from cache")
		return
	}
	if ri == nil {
		log.Debug("could not fine application profile definition in cache")
		return
	}
	appProfDef := management.ApplicationProfileDefinition{}
	if err := appProfDef.FromInstance(ri); err != nil {
		log.WithError(err).Error("error reading application profile definition from cache")
		return
	}

	updateDataFromEnumMap(data, appProfDef.Spec.Schema)
}

func (h *managedApplicationProfile) getManagedApp(_ context.Context, profile *management.ManagedApplicationProfile) (*management.ManagedApplication, error) {
	app := management.NewManagedApplication(profile.Spec.ManagedApplication, profile.Metadata.Scope.Name)
	ri, err := h.client.GetResource(app.GetSelfLink())
	if err != nil {
		return nil, err
	}

	app = &management.ManagedApplication{}
	err = app.FromInstance(ri)
	return app, err
}

// onError updates the managed app with an error status
func (h *managedApplicationProfile) onError(_ context.Context, profile *management.ManagedApplicationProfile, err error) {
	ps := prov.NewRequestStatusBuilder()
	status := ps.SetMessage(err.Error()).Failed()
	profile.Status = prov.NewStatusReason(status)
	profile.SubResources = map[string]interface{}{
		management.ManagedApplicationStatusSubResourceName: profile.Status,
	}
}

type provManagedAppProfile struct {
	managedAppName    string
	profileDefinition string
	teamName          string
	consumerOrgID     string
	id                string
	attributes        map[string]interface{}
	data              map[string]interface{}
}

// GetApplicationDetailsValue returns a value found on the managed application
func (a provManagedAppProfile) GetApplicationProfileData() map[string]interface{} {
	return a.attributes
}

// GetApplicationDetailsValue returns a value found on the managed application
func (a provManagedAppProfile) GetApplicationDetailsValue(key string) string {
	if a.data == nil {
		return ""
	}

	return util.ToString(a.data[key])
}

// GetManagedApplicationProfileName returns the name of the managed application
func (a provManagedAppProfile) GetManagedApplicationName() string {
	return a.managedAppName
}

// GetApplicationProfileName returns the name of the application profile definition
func (a provManagedAppProfile) GetApplicationProfileDefinitionName() string {
	return a.profileDefinition
}

// GetTeamName gets the owning team name for the managed application
func (a provManagedAppProfile) GetTeamName() string {
	return a.teamName
}

// GetConsumerOrgID returns the ID of the consumer org for the managed application
func (a provManagedAppProfile) GetConsumerOrgID() string {
	return a.consumerOrgID
}

// GetTeamName gets the owning team name for the managed application
func (a provManagedAppProfile) GetID() string {
	return a.id
}

func updateDataFromEnumMap(data map[string]interface{}, schema map[string]interface{}) {
	enumPropMap := prov.GetEnumValueMapsFromSchema(schema)
	if len(enumPropMap) == 0 {
		return
	}
	for k, l := range data {
		enumMap, ok := enumPropMap[k]
		if !ok {
			continue
		}
		label, ok := l.(string)
		if !ok {
			continue
		}
		value, ok := enumMap[label]
		if !ok {
			continue
		}
		data[k] = value
	}
}
