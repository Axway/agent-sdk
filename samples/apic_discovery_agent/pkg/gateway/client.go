package gateway

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/apic"
	"github.com/Axway/agent-sdk/pkg/util/log"

	// CHANGE_HERE - Change the import path(s) below to reference packages correctly
	"github.com/sbolosan/apic_discovery_agent/pkg/config"
)

// GatewayClient - Represents the Gateway client
type GatewayClient struct {
	cfg *config.GatewayConfig
}

// NewClient - Creates a new Gateway Client
func NewClient(gatewayCfg *config.GatewayConfig) (*GatewayClient, error) {
	return &GatewayClient{
		cfg: gatewayCfg,
	}, nil
}

// ExternalAPI - Sample struct representing the API definition in API gateway
type ExternalAPI struct {
	apiSpec       []byte
	id            string
	name          string
	description   string
	version       string
	url           string
	documentation []byte
}

// DiscoverAPIs - Process the API discovery
func (a *GatewayClient) DiscoverAPIs() error {
	// Gateway specific implementation to get the details for discovered API goes here
	// Set the service definition
	// As sample the implementation reads the spec definitions from local directory
	specFiles := a.listSpecFiles()
	for _, specFile := range specFiles {
		apiName, apiSpec, err := a.getSpec(specFile)
		if err != nil {
			log.Infof("Failed to load sample API specification from %s: %s ", a.cfg.SpecPath, err.Error())
		}
		externalAPI := ExternalAPI{
			id:            apiName,
			name:          apiName,
			description:   "Sample definition for API discovery agent",
			version:       "1.0.0",
			url:           "",
			documentation: []byte("\"Sample documentation for API discovery agent\""),
			apiSpec:       apiSpec,
		}

		serviceBody, err := a.buildServiceBody(externalAPI)
		if err != nil {
			return err
		}
		err = agent.PublishAPI(serviceBody)
		if err != nil {
			return err
		}
		log.Info("Published API " + serviceBody.APIName + "( type: " + serviceBody.ResourceType + " ) to AMPLIFY Central")
	}

	return nil
}

// buildServiceBody - creates the service definition
func (a *GatewayClient) buildServiceBody(externalAPI ExternalAPI) (apic.ServiceBody, error) {
	return apic.NewServiceBodyBuilder().
		SetID(externalAPI.id).
		SetAPIName(externalAPI.name).
		SetTitle(externalAPI.name).
		SetURL(externalAPI.url).
		SetDescription(externalAPI.description).
		SetAPISpec(externalAPI.apiSpec).
		SetVersion(externalAPI.version).
		SetAuthPolicy(apic.Passthrough).
		SetDocumentation(externalAPI.documentation).
		Build()
}

func (a *GatewayClient) getSpec(specFile string) (string, []byte, error) {
	fileName := filepath.Base(specFile)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))

	bytes, err := ioutil.ReadFile(specFile)
	if err != nil {
		return "", nil, err
	}
	return fileName, bytes, nil
}

func (a *GatewayClient) listSpecFiles() []string {
	var files []string
	err := filepath.Walk(a.cfg.SpecPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	return files
}
