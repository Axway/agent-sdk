package apic

import (
	"errors"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
)

var typeToComponent = map[string]map[string]string{
	GitLab.String(): {
		management.DiscoveryAgentGVK().Kind: "gitlab-discovery-agent",
	},
	Backstage.String(): {
		management.DiscoveryAgentGVK().Kind: "backstage-discovery-agent",
	},
	Akamai.String(): {
		management.TraceabilityAgentGVK().Kind: "akamai-agent",
	},
	APIConnect.String(): {
		management.DiscoveryAgentGVK().Kind:    "ibm-apiconnect-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "ibm-apiconnect-traceability-agent",
	},
	Apigee.String(): {
		management.DiscoveryAgentGVK().Kind:    "apigee-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "apigee-traceability-agent",
	},
	APIM.String(): {
		management.DiscoveryAgentGVK().Kind:    "v7-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "v7-traceability-agent",
	},
	AWS.String(): {
		management.DiscoveryAgentGVK().Kind:    "aws-apigw-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "aws-apigw-traceability-agent",
	},
	Azure.String(): {
		management.DiscoveryAgentGVK().Kind:    "azure-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "azure-traceability-agent",
	},
	Istio.String(): {
		management.DiscoveryAgentGVK().Kind:    "istio-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "als-traceability-agent",
	},
	Kafka.String(): {
		management.DiscoveryAgentGVK().Kind:    "kafka-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "kafka-traceability-agent",
	},
	WebMethods.String(): {
		management.DiscoveryAgentGVK().Kind:    "software-ag-webmethods-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "software-ag-webmethods-traceability-agent",
	},
	Kong.String(): {
		management.DiscoveryAgentGVK().Kind:    "kong-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "kong-traceability-agent",
	},
	Mulesoft.String(): {
		management.DiscoveryAgentGVK().Kind:    "mulesoft-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "mulesoft-traceability-agent",
	},
	SAPAPIPortal.String(): {
		management.DiscoveryAgentGVK().Kind:    "sap-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "sap-traceability-agent",
	},

	Graylog.String(): {
		management.TraceabilityAgentGVK().Kind: "graylog-agent",
	},
	Traceable.String(): {
		management.TraceabilityAgentGVK().Kind: "traceable-agent",
	},
	WSO2.String(): {
		management.DiscoveryAgentGVK().Kind:    "wso2-discovery-agent",
		management.TraceabilityAgentGVK().Kind: "wso2-traceability-agent",
	},
}

func GetComponent(dataplaneType, agentResourceKind string) (string, error) {
	componentTypeMap, ok := typeToComponent[dataplaneType]
	if !ok {
		return "", errors.New("could not find dataplane type")
	}

	component, ok := componentTypeMap[agentResourceKind]
	if !ok {
		return "", errors.New("could not find component name from dataplane type and agent kind")
	}
	return component, nil
}
