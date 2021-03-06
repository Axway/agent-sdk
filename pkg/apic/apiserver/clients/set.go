/*
 * This file is automatically generated
 */

package clients

import (
	"fmt"

	cAPIV1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/api/v1"
	catalog_v1alpha1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1"
	definitions_v1alpha1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1"
	management_v1alpha1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1"
)

type Set struct {
	DiscoveryAgentManagementV1alpha1                 *management_v1alpha1.UnscopedDiscoveryAgentClient
	TraceabilityAgentManagementV1alpha1              *management_v1alpha1.UnscopedTraceabilityAgentClient
	GovernanceAgentManagementV1alpha1                *management_v1alpha1.UnscopedGovernanceAgentClient
	EnvironmentManagementV1alpha1                    *management_v1alpha1.EnvironmentClient
	APIServiceManagementV1alpha1                     *management_v1alpha1.UnscopedAPIServiceClient
	APIServiceRevisionManagementV1alpha1             *management_v1alpha1.UnscopedAPIServiceRevisionClient
	APIServiceInstanceManagementV1alpha1             *management_v1alpha1.UnscopedAPIServiceInstanceClient
	ConsumerInstanceManagementV1alpha1               *management_v1alpha1.UnscopedConsumerInstanceClient
	ConsumerSubscriptionDefinitionManagementV1alpha1 *management_v1alpha1.UnscopedConsumerSubscriptionDefinitionClient
	IntegrationManagementV1alpha1                    *management_v1alpha1.IntegrationClient
	ResourceHookManagementV1alpha1                   *management_v1alpha1.UnscopedResourceHookClient
	K8SClusterManagementV1alpha1                     *management_v1alpha1.K8SClusterClient
	K8SResourceManagementV1alpha1                    *management_v1alpha1.UnscopedK8SResourceClient
	ResourceDiscoveryManagementV1alpha1              *management_v1alpha1.UnscopedResourceDiscoveryClient
	MeshManagementV1alpha1                           *management_v1alpha1.MeshClient
	SpecDiscoveryManagementV1alpha1                  *management_v1alpha1.UnscopedSpecDiscoveryClient
	APISpecManagementV1alpha1                        *management_v1alpha1.UnscopedAPISpecClient
	MeshWorkloadManagementV1alpha1                   *management_v1alpha1.UnscopedMeshWorkloadClient
	MeshServiceManagementV1alpha1                    *management_v1alpha1.UnscopedMeshServiceClient
	MeshDiscoveryManagementV1alpha1                  *management_v1alpha1.UnscopedMeshDiscoveryClient
	AssetMappingTemplateManagementV1alpha1           *management_v1alpha1.UnscopedAssetMappingTemplateClient
	AssetMappingManagementV1alpha1                   *management_v1alpha1.UnscopedAssetMappingClient
	AccessRequestDefinitionManagementV1alpha1        *management_v1alpha1.UnscopedAccessRequestDefinitionClient
	AccessRequestManagementV1alpha1                  *management_v1alpha1.UnscopedAccessRequestClient
	DeploymentManagementV1alpha1                     *management_v1alpha1.UnscopedDeploymentClient
	VirtualAPIManagementV1alpha1                     *management_v1alpha1.VirtualAPIClient
	VirtualAPIReleaseManagementV1alpha1              *management_v1alpha1.VirtualAPIReleaseClient
	ReleaseTagManagementV1alpha1                     *management_v1alpha1.UnscopedReleaseTagClient
	VirtualServiceManagementV1alpha1                 *management_v1alpha1.UnscopedVirtualServiceClient
	CorsRuleManagementV1alpha1                       *management_v1alpha1.UnscopedCorsRuleClient
	OAS3DocumentManagementV1alpha1                   *management_v1alpha1.UnscopedOAS3DocumentClient
	WebhookManagementV1alpha1                        *management_v1alpha1.UnscopedWebhookClient
	SecretManagementV1alpha1                         *management_v1alpha1.UnscopedSecretClient
	StageCatalogV1alpha1                             *catalog_v1alpha1.StageClient
	AssetCatalogV1alpha1                             *catalog_v1alpha1.AssetClient
	AssetReleaseCatalogV1alpha1                      *catalog_v1alpha1.AssetReleaseClient
	CategoryCatalogV1alpha1                          *catalog_v1alpha1.CategoryClient
	ProductCatalogV1alpha1                           *catalog_v1alpha1.ProductClient
	ReleaseTagCatalogV1alpha1                        *catalog_v1alpha1.UnscopedReleaseTagClient
	AssetResourceCatalogV1alpha1                     *catalog_v1alpha1.UnscopedAssetResourceClient
	AssetRequestDefinitionCatalogV1alpha1            *catalog_v1alpha1.UnscopedAssetRequestDefinitionClient
	AssetRequestCatalogV1alpha1                      *catalog_v1alpha1.UnscopedAssetRequestClient
	DocumentCatalogV1alpha1                          *catalog_v1alpha1.UnscopedDocumentClient
	ResourceGroupDefinitionsV1alpha1                 *definitions_v1alpha1.ResourceGroupClient
	ResourceDefinitionDefinitionsV1alpha1            *definitions_v1alpha1.UnscopedResourceDefinitionClient
	ResourceDefinitionVersionDefinitionsV1alpha1     *definitions_v1alpha1.UnscopedResourceDefinitionVersionClient
	CommandLineInterfaceDefinitionsV1alpha1          *definitions_v1alpha1.UnscopedCommandLineInterfaceClient
}

func New(b cAPIV1.Base) *Set {
	s := &Set{}

	var err error

	s.DiscoveryAgentManagementV1alpha1, err = management_v1alpha1.NewDiscoveryAgentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.DiscoveryAgent: %s", err))
	}
	s.TraceabilityAgentManagementV1alpha1, err = management_v1alpha1.NewTraceabilityAgentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.TraceabilityAgent: %s", err))
	}
	s.GovernanceAgentManagementV1alpha1, err = management_v1alpha1.NewGovernanceAgentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.GovernanceAgent: %s", err))
	}
	s.EnvironmentManagementV1alpha1, err = management_v1alpha1.NewEnvironmentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Environment: %s", err))
	}
	s.APIServiceManagementV1alpha1, err = management_v1alpha1.NewAPIServiceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.APIService: %s", err))
	}
	s.APIServiceRevisionManagementV1alpha1, err = management_v1alpha1.NewAPIServiceRevisionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.APIServiceRevision: %s", err))
	}
	s.APIServiceInstanceManagementV1alpha1, err = management_v1alpha1.NewAPIServiceInstanceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.APIServiceInstance: %s", err))
	}
	s.ConsumerInstanceManagementV1alpha1, err = management_v1alpha1.NewConsumerInstanceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.ConsumerInstance: %s", err))
	}
	s.ConsumerSubscriptionDefinitionManagementV1alpha1, err = management_v1alpha1.NewConsumerSubscriptionDefinitionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.ConsumerSubscriptionDefinition: %s", err))
	}
	s.IntegrationManagementV1alpha1, err = management_v1alpha1.NewIntegrationClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Integration: %s", err))
	}
	s.ResourceHookManagementV1alpha1, err = management_v1alpha1.NewResourceHookClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.ResourceHook: %s", err))
	}
	s.K8SClusterManagementV1alpha1, err = management_v1alpha1.NewK8SClusterClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.K8SCluster: %s", err))
	}
	s.K8SResourceManagementV1alpha1, err = management_v1alpha1.NewK8SResourceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.K8SResource: %s", err))
	}
	s.ResourceDiscoveryManagementV1alpha1, err = management_v1alpha1.NewResourceDiscoveryClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.ResourceDiscovery: %s", err))
	}
	s.MeshManagementV1alpha1, err = management_v1alpha1.NewMeshClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Mesh: %s", err))
	}
	s.SpecDiscoveryManagementV1alpha1, err = management_v1alpha1.NewSpecDiscoveryClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.SpecDiscovery: %s", err))
	}
	s.APISpecManagementV1alpha1, err = management_v1alpha1.NewAPISpecClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.APISpec: %s", err))
	}
	s.MeshWorkloadManagementV1alpha1, err = management_v1alpha1.NewMeshWorkloadClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.MeshWorkload: %s", err))
	}
	s.MeshServiceManagementV1alpha1, err = management_v1alpha1.NewMeshServiceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.MeshService: %s", err))
	}
	s.MeshDiscoveryManagementV1alpha1, err = management_v1alpha1.NewMeshDiscoveryClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.MeshDiscovery: %s", err))
	}
	s.AssetMappingTemplateManagementV1alpha1, err = management_v1alpha1.NewAssetMappingTemplateClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.AssetMappingTemplate: %s", err))
	}
	s.AssetMappingManagementV1alpha1, err = management_v1alpha1.NewAssetMappingClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.AssetMapping: %s", err))
	}
	s.AccessRequestDefinitionManagementV1alpha1, err = management_v1alpha1.NewAccessRequestDefinitionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.AccessRequestDefinition: %s", err))
	}
	s.AccessRequestManagementV1alpha1, err = management_v1alpha1.NewAccessRequestClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.AccessRequest: %s", err))
	}
	s.DeploymentManagementV1alpha1, err = management_v1alpha1.NewDeploymentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Deployment: %s", err))
	}
	s.VirtualAPIManagementV1alpha1, err = management_v1alpha1.NewVirtualAPIClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.VirtualAPI: %s", err))
	}
	s.VirtualAPIReleaseManagementV1alpha1, err = management_v1alpha1.NewVirtualAPIReleaseClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.VirtualAPIRelease: %s", err))
	}
	s.ReleaseTagManagementV1alpha1, err = management_v1alpha1.NewReleaseTagClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.ReleaseTag: %s", err))
	}
	s.VirtualServiceManagementV1alpha1, err = management_v1alpha1.NewVirtualServiceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.VirtualService: %s", err))
	}
	s.CorsRuleManagementV1alpha1, err = management_v1alpha1.NewCorsRuleClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.CorsRule: %s", err))
	}
	s.OAS3DocumentManagementV1alpha1, err = management_v1alpha1.NewOAS3DocumentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.OAS3Document: %s", err))
	}
	s.WebhookManagementV1alpha1, err = management_v1alpha1.NewWebhookClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Webhook: %s", err))
	}
	s.SecretManagementV1alpha1, err = management_v1alpha1.NewSecretClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/management/v1alpha1.Secret: %s", err))
	}
	s.StageCatalogV1alpha1, err = catalog_v1alpha1.NewStageClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.Stage: %s", err))
	}
	s.AssetCatalogV1alpha1, err = catalog_v1alpha1.NewAssetClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.Asset: %s", err))
	}
	s.AssetReleaseCatalogV1alpha1, err = catalog_v1alpha1.NewAssetReleaseClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.AssetRelease: %s", err))
	}
	s.CategoryCatalogV1alpha1, err = catalog_v1alpha1.NewCategoryClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.Category: %s", err))
	}
	s.ProductCatalogV1alpha1, err = catalog_v1alpha1.NewProductClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.Product: %s", err))
	}
	s.ReleaseTagCatalogV1alpha1, err = catalog_v1alpha1.NewReleaseTagClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.ReleaseTag: %s", err))
	}
	s.AssetResourceCatalogV1alpha1, err = catalog_v1alpha1.NewAssetResourceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.AssetResource: %s", err))
	}
	s.AssetRequestDefinitionCatalogV1alpha1, err = catalog_v1alpha1.NewAssetRequestDefinitionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.AssetRequestDefinition: %s", err))
	}
	s.AssetRequestCatalogV1alpha1, err = catalog_v1alpha1.NewAssetRequestClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.AssetRequest: %s", err))
	}
	s.DocumentCatalogV1alpha1, err = catalog_v1alpha1.NewDocumentClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/catalog/v1alpha1.Document: %s", err))
	}
	s.ResourceGroupDefinitionsV1alpha1, err = definitions_v1alpha1.NewResourceGroupClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1.ResourceGroup: %s", err))
	}
	s.ResourceDefinitionDefinitionsV1alpha1, err = definitions_v1alpha1.NewResourceDefinitionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1.ResourceDefinition: %s", err))
	}
	s.ResourceDefinitionVersionDefinitionsV1alpha1, err = definitions_v1alpha1.NewResourceDefinitionVersionClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1.ResourceDefinitionVersion: %s", err))
	}
	s.CommandLineInterfaceDefinitionsV1alpha1, err = definitions_v1alpha1.NewCommandLineInterfaceClient(b)
	if err != nil {
		panic(fmt.Sprintf("Failed to create client for github.com/Axway/agent-sdk/pkg/apic/apiserver/clients/definitions/v1alpha1.CommandLineInterface: %s", err))
	}
	return s
}
