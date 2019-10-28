package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/auth"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	log "github.com/sirupsen/logrus"
)

var tokenRequester *apicauth.PlatformTokenGetter
var httpClient = http.DefaultClient

func init() {
	tokenURL := auth.GetAuthConfig().GetTokenURL()
	aud := auth.GetAuthConfig().GetRealmURL()
	priKey := auth.GetAuthConfig().GetPrivateKey()
	pubKey := auth.GetAuthConfig().GetPublicKey()
	keyPwd := auth.GetAuthConfig().GetKeyPwd()
	clientID := auth.GetAuthConfig().GetClientID()
	authTimeout := auth.GetAuthConfig().GetAuthTimeout()
	tokenRequester = apicauth.NewPlatformTokenGetter(priKey, pubKey, keyPwd, tokenURL, aud, clientID, authTimeout)
}

// DeployAPI -
func DeployAPI(method string, apiServerBuffer []byte, agentMode string, url string) (string, error) {
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

func handleResponse(method string, agentMode string, detail map[string]*json.RawMessage) (string, error) {
	if strings.ToLower(agentMode) == strings.ToLower("catalog") {

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

	request.Header.Add("X-Axway-Tenant-Id", apicConfig.GetTenantID())
	request.Header.Add("Authorization", "Bearer "+token)
	request.Header.Add("Content-Type", "application/json")
	return request, nil
}
