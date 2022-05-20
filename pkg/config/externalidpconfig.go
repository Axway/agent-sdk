package config

import (
	"encoding/json"
	"strconv"

	"github.com/Axway/agent-sdk/pkg/cmd/properties"
)

const (
	pathExternalIDP = "agentFeatures.idp"
)

// ExternalIDPConfig -
type ExternalIDPConfig interface {
	GetIDPList() []IDPConfig
}

type externalIDPConfig struct {
	IDPConfigs map[string]IDPConfig
}

func (e *externalIDPConfig) GetIDPList() []IDPConfig {
	list := make([]IDPConfig, 0)
	for _, idpCfg := range e.IDPConfigs {
		list = append(list, idpCfg)
	}
	return list
}

// ExtraProperties - type for representing extra IdP provider properties to be included in client request
type ExtraProperties map[string]string

// UnmarshalJSON - deserializes extra properties from env config
func (e *ExtraProperties) UnmarshalJSON(data []byte) error {
	m := make(map[string]string)
	buf, _ := strconv.Unquote(string(data))
	json.Unmarshal([]byte(buf), &m)

	em := map[string]string(*e)
	for key, val := range m {
		em[key] = val
	}

	return nil
}

// IDPConfig - interface for IdP provider config
type IDPConfig interface {
	GetMetadataURL() string
	GetIDPType() string
	GetIDPName() string
	GetAccessToken() string
	GetExtraProperties() map[string]string
}

// IDPConfiguration - Structure to hold the IdP provider config
type IDPConfiguration struct {
	Name            string          `json:"name,omitempty"`
	Type            string          `json:"type,omitempty"`
	MetadataURL     string          `json:"metadataUrl,omitempty"`
	AccessToken     string          `json:"accessToken,omitempty"`
	ExtraProperties ExtraProperties `json:"extraProperties,omitempty"`
}

// GetIDPName - returns the name of IdP provider
func (i *IDPConfiguration) GetIDPName() string {
	return i.Name
}

// GetIDPType - returns the IdP type
func (i *IDPConfiguration) GetIDPType() string {
	return i.Type
}

// GetMetadataURL - returns the metadata URL for IdP
func (i *IDPConfiguration) GetMetadataURL() string {
	return i.MetadataURL
}

// GetAccessToken - returns the access token to be used for IdP client registration APIs
func (i *IDPConfiguration) GetAccessToken() string {
	return i.AccessToken
}

// GetExtraProperties - returns the IdP specific properties to be included in client request
func (i *IDPConfiguration) GetExtraProperties() map[string]string {
	return i.ExtraProperties
}

func addExternalIDPProperties(props properties.Properties) {
	props.AddObjectSliceProperty(pathExternalIDP, []string{"name", "type", "metadataUrl", "accessToken", "extraProperties"})
}

func parseExternalIDPConfig(props properties.Properties) (ExternalIDPConfig, error) {
	envIDPCfgList := props.ObjectSlicePropertyValue(pathExternalIDP)

	cfg := &externalIDPConfig{
		IDPConfigs: make(map[string]IDPConfig),
	}

	for _, envIdpCfg := range envIDPCfgList {
		idpCfg := &IDPConfiguration{
			ExtraProperties: make(ExtraProperties),
		}

		buf, _ := json.Marshal(envIdpCfg)
		json.Unmarshal(buf, idpCfg)

		cfg.IDPConfigs[idpCfg.Name] = idpCfg
	}

	return cfg, nil
}
