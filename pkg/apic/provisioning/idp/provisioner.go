package idp

import (
	"context"
	"fmt"
	"strings"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1"
	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/Axway/agent-sdk/pkg/authz/oauth"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/Axway/agent-sdk/pkg/util"
)

const (
	registrationClientURIKey = "registrationClientURI"
)

type Provisioner interface {
	IsIDPCredential() bool
	GetIDPProvider() oauth.Provider
	GetIDPCredentialData() provisioning.IDPCredentialData
	RegisterClient() error
	UnregisterClient() error
	GetAgentDetails() (map[string]string, error)
	Validate() error
}

type provisioner struct {
	app              *management.ManagedApplication
	credential       *management.Credential
	idpProvider      oauth.Provider
	credentialData   *credData
	agentDetail      map[string]string
	provisioningMode string
}

type ProvisionerOption func(p *provisioner)

func NewProvisioner(ctx context.Context, idpRegistry oauth.IdPRegistry, app *management.ManagedApplication, credential *management.Credential, opts ...oauth.ConfigOption) (Provisioner, error) {
	p := &provisioner{
		app:            app,
		credential:     credential,
		credentialData: &credData{},
		agentDetail:    make(map[string]string),
	}
	if credential.Spec.Provision != nil {
		p.provisioningMode = credential.Spec.Provision.Mode
	}

	idpTokenURL, ok := p.credential.Spec.Data[provisioning.IDPTokenURL].(string)
	if ok && idpRegistry != nil {
		idpProvider, err := idpRegistry.GetProviderByTokenEndpoint(ctx, idpTokenURL, opts...)
		if err != nil {
			return nil, err
		}

		if idpProvider != nil {
			p.idpProvider = idpProvider
		}
	}

	if p.idpProvider != nil || p.provisioningMode == provisioning.CredProvisionModeExternal {
		p.initCredentialData()
	}
	return p, nil
}

func getProvisionedData(cred *management.Credential) map[string]interface{} {
	var provData map[string]interface{}
	if cred.Data != nil {
		if m, ok := cred.Data.(map[string]interface{}); ok {
			provData = m
		}
	}
	return provData
}

func (p *provisioner) initCredentialData() {
	provData := getProvisionedData(p.credential)
	if provData != nil {
		p.credentialData.clientID = util.GetStringFromMapInterface(provisioning.OauthClientID, provData)
	}
	credData := p.credential.Spec.Data

	p.credentialData.scopes = util.GetStringArrayFromMapInterface(provisioning.OauthScopes, credData)
	p.credentialData.grantTypes = []string{util.GetStringFromMapInterface(provisioning.OauthGrantType, credData)}
	p.credentialData.redirectURLs = util.GetStringArrayFromMapInterface(provisioning.OauthRedirectURIs, credData)
	p.credentialData.tokenAuthMethod = util.GetStringFromMapInterface(provisioning.OauthTokenAuthMethod, credData)
	p.credentialData.publicKey = util.GetStringFromMapInterface(provisioning.OauthJwks, credData)
	p.credentialData.jwksURI = util.GetStringFromMapInterface(provisioning.OauthJwksURI, credData)
	p.credentialData.certificate = util.GetStringFromMapInterface(provisioning.OauthCertificate, credData)
	p.credentialData.certificateMetadata = util.GetStringFromMapInterface(provisioning.OauthCertificateMetadata, credData)
	p.credentialData.tlsClientAuthSanDNS = util.GetStringFromMapInterface(provisioning.OauthTLSAuthSANDNS, credData)
	p.credentialData.tlsClientAuthSanEmail = util.GetStringFromMapInterface(provisioning.OauthTLSAuthSANEmail, credData)
	p.credentialData.tlsClientAuthSanIP = util.GetStringFromMapInterface(provisioning.OauthTLSAuthSANIP, credData)
	p.credentialData.tlsClientAuthSanURI = util.GetStringFromMapInterface(provisioning.OauthTLSAuthSANURI, credData)
	registrationToken := p.getRegistrationTokenFromAgentDetails()
	if registrationToken != "" {
		p.decryptRegistrationToken(registrationToken)
	}
}

func (p *provisioner) IsIDPCredential() bool {
	return p.idpProvider != nil
}

func (p *provisioner) GetIDPProvider() oauth.Provider {
	return p.idpProvider
}

func (p *provisioner) GetIDPCredentialData() provisioning.IDPCredentialData {
	return p.credentialData
}

func (p *provisioner) RegisterClient() error {
	if !p.IsIDPCredential() {
		return nil
	}

	clientName, err := p.appClientName()
	if err != nil {
		return err
	}

	builder := oauth.NewClientMetadataBuilder().
		SetClientName(clientName).
		SetScopes(p.credentialData.GetScopes()).
		SetGrantTypes(p.credentialData.GetGrantTypes()).
		SetTokenEndpointAuthMethod(p.credentialData.GetTokenEndpointAuthMethod()).
		SetResponseType(p.credentialData.GetResponseTypes()).
		SetRedirectURIs(p.credentialData.GetRedirectURIs())

	if p.credentialData.GetTokenEndpointAuthMethod() == config.PrivateKeyJWT {
		builder.SetJWKS([]byte(formattedJWKS(p.credentialData.GetPublicKey()))).
			SetJWKSURI(p.credentialData.GetJwksURI())
	}

	if p.credentialData.GetTokenEndpointAuthMethod() == config.TLSClientAuth || p.credentialData.GetTokenEndpointAuthMethod() == config.SelfSignedTLSClientAuth {
		builder.SetJWKS([]byte(formattedJWKS(p.credentialData.GetCertificate()))).
			SetCertificateMetadata(p.credentialData.GetCertificateMetadata()).
			SetTLSClientAuthSanDNS(p.credentialData.GetTLSClientAuthSanDNS()).
			SetTLSClientAuthSanEmail(p.credentialData.GetTLSClientAuthSanEmail()).
			SetTLSClientAuthSanIP(p.credentialData.GetTLSClientAuthSanIP()).
			SetTLSClientAuthSanURI(p.credentialData.GetTLSClientAuthSanURI())
	}

	clientMetadata, err := builder.Build()
	if err != nil {
		return err
	}

	resClientMetadata, err := p.idpProvider.RegisterClient(clientMetadata)
	if err != nil {
		return err
	}

	p.credentialData.registrationAccessToken = resClientMetadata.GetRegistrationAccessToken()
	p.credentialData.clientID = resClientMetadata.GetClientID()
	p.credentialData.clientSecret = resClientMetadata.GetClientSecret()

	if resClientMetadata.GetRegistrationClientURI() != "" {
		util.SetAgentDetailsKey(p.credential, registrationClientURIKey, resClientMetadata.GetRegistrationClientURI())
	}

	return nil
}

func (p *provisioner) UnregisterClient() error {
	if !p.IsIDPCredential() {
		return nil
	}

	registrationClientURI, _ := util.GetAgentDetailsValue(p.credential, registrationClientURIKey)

	scopes := p.credentialData.GetScopes()
	grantType := ""
	if gt := p.credentialData.GetGrantTypes(); len(gt) > 0 {
		grantType = gt[0]
	}

	err := p.idpProvider.UnregisterClient(p.credentialData.GetClientID(), p.credentialData.registrationAccessToken, registrationClientURI, scopes, grantType)
	if err != nil {
		return err
	}

	p.credentialData.clientID = p.credentialData.GetClientID()
	return nil
}

func (p *provisioner) GetAgentDetails() (map[string]string, error) {
	registrationToken, err := p.encryptRegistrationToken()
	if err != nil {
		return nil, err
	}
	return p.createAgentDetails(registrationToken), nil
}

func (p *provisioner) Validate() error {
	if !p.IsIDPCredential() {
		return nil
	}

	return p.idpProvider.Validate()
}

func (p *provisioner) encryptRegistrationToken() (string, error) {
	if p.credentialData.registrationAccessToken != "" {
		enc, err := util.NewGCMEncryptor([]byte(p.app.Spec.Security.EncryptionKey))
		if err != nil {
			return "", err
		}

		ert, err := enc.Encrypt(p.credentialData.registrationAccessToken)
		if err != nil {
			return "", err
		}
		return ert, nil
	}
	return "", nil
}

func (p *provisioner) decryptRegistrationToken(encryptedToken string) error {
	if encryptedToken != "" {
		dc, err := util.NewGCMDecryptor([]byte(p.app.Spec.Security.EncryptionKey))
		if err != nil {
			return err
		}

		decrypted, err := dc.Decrypt(encryptedToken)
		if err != nil {
			return err
		}
		p.credentialData.registrationAccessToken = decrypted
	}
	return nil
}

func (p *provisioner) createAgentDetails(registrationToken string) map[string]string {
	agentDetail := make(map[string]string)
	if registrationToken != "" {
		agentDetail[provisioning.OauthRegistrationToken] = registrationToken
	}
	return agentDetail
}

func (p *provisioner) getRegistrationTokenFromAgentDetails() string {
	registrationToken, _ := util.GetAgentDetailsValue(p.credential, provisioning.OauthRegistrationToken)
	return registrationToken
}

func (p *provisioner) appClientName() (string, error) {
	idpCfg := p.idpProvider.GetConfig()
	if idpCfg == nil || idpCfg.GetIDPType() != oauth.TypeOkta {
		return p.credential.GetName(), nil
	}
	oktaCfg, ok := idpCfg.(interface{ GetAppNameTemplate() string })
	if !ok {
		return p.credential.GetName(), nil
	}

	template := oktaCfg.GetAppNameTemplate()
	teamName, _ := util.GetAgentDetailsValue(p.app, provisioning.AgentDetailTeamName)
	name := strings.NewReplacer(
		config.OktaPlaceholderMPApplicationName, p.app.Name,
		config.OktaPlaceholderOwningTeam, teamName,
		config.OktaPlaceholderCredentialName, p.credential.GetName(),
	).Replace(template)

	name = util.NormalizeNameForCentral(name)
	if len(name) > 100 {
		return "", fmt.Errorf("Okta app name exceeds 100-character limit after normalization: %d chars", len(name))
	}
	return name, nil
}
