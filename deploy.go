package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	corecfg "git.ecd.axway.int/apigov/aws_apigw_discovery_agent/core/config"
	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/config"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	"github.com/sirupsen/logrus"
)

var tokenRequester *apicauth.PlatformTokenGetter
var httpClient = http.DefaultClient

var log logrus.FieldLogger = logrus.WithField("package", "apic")

// SetLog sets the logger for the package.
func SetLog(newLog logrus.FieldLogger) {
	log = newLog
	return
}

// Initialize -
func Initialize() {
	tokenURL := config.GetConfig().CentralConfig.GetAuthConfig().GetTokenURL()
	aud := config.GetConfig().CentralConfig.GetAuthConfig().GetAudience()
	priKey := config.GetConfig().CentralConfig.GetAuthConfig().GetPrivateKey()
	pubKey := config.GetConfig().CentralConfig.GetAuthConfig().GetPublicKey()
	keyPwd := config.GetConfig().CentralConfig.GetAuthConfig().GetKeyPassword()
	clientID := config.GetConfig().CentralConfig.GetAuthConfig().GetClientID()
	authTimeout := config.GetConfig().CentralConfig.GetAuthConfig().GetTimeout()
	tokenRequester = apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout)
}

// DeployAPI -
func DeployAPI(method string, apiServerBuffer []byte, agentMode corecfg.AgentMode, url string) (string, error) {
	request, err := setHeader(method, url, bytes.NewBuffer(apiServerBuffer))
	if err != nil {
		return "", err
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return "", err
	}
	detail := make(map[string]*json.RawMessage)
	if !(response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated) {

		json.NewDecoder(response.Body).Decode(&detail)
		for k, v := range detail {
			buffer, _ := v.MarshalJSON()
			log.Debugf("HTTP response key %v: %v", k, string(buffer))
		}
		return "", errors.New(response.Status)
	}
	defer response.Body.Close()
	json.NewDecoder(response.Body).Decode(&detail)

	return handleResponse(method, agentMode, detail)
}

func handleResponse(method string, agentMode corecfg.AgentMode, detail map[string]*json.RawMessage) (string, error) {
	if agentMode != corecfg.Connected {
		if strings.ToLower(method) == strings.ToLower("POST") {
			itemID := ""
			for k, v := range detail {
				buffer, _ := v.MarshalJSON()
				if k == "id" {
					itemID = string(buffer)
				}
				log.Debugf("HTTP response key %v: %v", k, string(buffer))
			}
			return strconv.Unquote(itemID)
		}
		// This is an update to catalog item (PUT)
		for k, v := range detail {
			buffer, _ := v.MarshalJSON()
			log.Debugf("HTTP response key %v: %v", k, string(buffer))
		}

	}

	return "", nil

}

func setHeader(method, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	var token string
	if token, err = tokenRequester.GetToken(); err != nil {
		return nil, err
	}

	request.Header.Add("X-Axway-Tenant-Id", config.GetConfig().CentralConfig.GetTenantID())
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Content-Type", "application/json")
	return request, nil
}
