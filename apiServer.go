package apic

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"git.ecd.axway.int/apigov/aws_apigw_discovery_agent/pkg/auth"
	"git.ecd.axway.int/apigov/service-mesh-agent/pkg/apicauth"
	log "github.com/sirupsen/logrus"
)

// APIServer -
type APIServer struct {
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Spec       map[string]interface{} `json:"spec"`
}

// Spec -
type Spec struct {
	Description string `json:"description"`
}

var apiServerTokenRequester *apicauth.PlatformTokenGetter

var apiServerHTTPClient = http.DefaultClient

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

const apiServerSubscriptionSchema = "{\"type\": \"object\", \"$schema\": \"http://json-schema.org/draft-04/schema#\", \"description\": \"Subscription specification for API Key authentication\", \"x-axway-unique-keys\": \"APIC_APPLICATION_ID\", \"properties\": {\"applicationId\": {\"type\": \"string\", \"description\": \"Select an application\", \"x-axway-ref-apic\": \"APIC_APPLICATION_ID\"}}, \"required\":[\"applicationId\"]}"

// CreateAPIServerBodyForAdd -
func CreateAPIServerBodyForAdd(apiID, apiName, stageName string, stageTags []string) ([]byte, error) {

	// TODO - Hardcoding attributes for now.  We can use this for whatever
	attribute := make(map[string]interface{})
	attribute["release"] = "1.1.0"
	attribute["stage"] = "abc"
	attribute["gas"] = "beano"

	// spec needs to adhere to environment schema
	spec := make(map[string]interface{})
	spec["description"] = "API From AWS APIGateway (RestApiId: " + apiID + ", StageName: " + stageName + ")"

	apiServerService := APIServer{
		Name:       strings.ToLower(apiName), // name needs to be path friendly and follows this regex "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\
		Title:      fmt.Sprintf("%v (Stage: %v)", apiName, stageName),
		Attributes: attribute,
		Spec:       spec,
		Tags:       stageTags,
	}

	return json.Marshal(apiServerService)
}

// AddAPIServer -
func AddAPIServer(apiServerBuffer []byte, apiServerEnv string) (string, error) {

	// local
	request, err := apiServerServiceRequest("POST", "http://localhost:8080/apis/management/v1alpha1/environments/"+apiServerEnv+"/apiservices", bytes.NewBuffer(apiServerBuffer))

	// seby's namespace
	// request, err := apiServerServiceRequest("POST", "http://beta.xenon.apicentral-k8s.axwaytest.net/apis/management/v1alpha1/environments/"+apiServerEnv+"/apiservices", bytes.NewBuffer(apiServerBuffer))

	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/json")

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

	// TODO
	// Need to figure out if this is really needed.  API server service can update Revision, but not sure if its equivalent to catalogItem
	// itemID := ""
	// for k, v := range detail {
	// 	buffer, _ := v.MarshalJSON()
	// 	if k == "id" {
	// 		itemID = string(buffer)
	// 	}
	// 	log.Debugf("HTTP response key %v: %v", k, string(buffer))
	// }
	// return strconv.Unquote(itemID)

	return "", err
}

func apiServerServiceRequest(method, url string, body io.Reader) (*http.Request, error) {
	request, err := http.NewRequest(method, url, body)
	var token string
	if token, err = tokenRequester.GetToken(); err != nil {
		return nil, err
	}

	// ran on local server
	request.Header.Add("X-Axway-Tenant-Id", "axway")
	request.Header.Add("Authorization", "Bearer "+token)

	// seby's namespace
	// token = "eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJqdGkiOiIyM2ZiZWM4MS05ZGEyLTQzNTEtOWE3My02ODEyOTVkYzlhMmMiLCJleHAiOjE1NzE3ODA4NTYsIm5iZiI6MCwiaWF0IjoxNTcxNzc3MjU2LCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiI3MjNlYTJkNC05OGU4LTQ1MGItOWZkOC0xMDM4NDJjYmZkM2YiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwiYXV0aF90aW1lIjoxNTcxNzc2MDk4LCJzZXNzaW9uX3N0YXRlIjoiODg5OGY1MTMtNmRlMy00ZmI3LTgxZjktODI2ZTkzZjQzZWFmIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJvZmZsaW5lX2FjY2VzcyIsInVtYV9hdXRob3JpemF0aW9uIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYWNjb3VudCI6eyJyb2xlcyI6WyJtYW5hZ2UtYWNjb3VudCIsIm1hbmFnZS1hY2NvdW50LWxpbmtzIiwidmlldy1wcm9maWxlIl19fSwic2NvcGUiOiIiLCJzdWIiOiJlYWM2ZjlmNS1kYmZhLTRhODctOThmNC1jM2JjMjA5Nzc2NzgiLCJpZGVudGl0eV9wcm92aWRlciI6IjM2MCIsImlzcyI6Imh0dHBzOi8vbG9naW4tcHJlcHJvZC5heHdheS5jb20vYXV0aC9yZWFsbXMvQnJva2VyIiwicHJlZmVycmVkX3VzZXJuYW1lIjoibW9nb3Muc2ViYXN0aWFuK3Rlc3RAZ21haWwuY29tIiwiZW1haWwiOiJtb2dvcy5zZWJhc3RpYW4rdGVzdEBnbWFpbC5jb20ifQ.AoNiWmlus2SwlS8Toezt5g2rmOlaJevhn33lygxXS3M44zEwUzK_oUPFI4MUvmhhjNMX6ZqgWmSuKseOUi0rbCkDiRWDZ95Dr2AhuddGS32Lcu4Axv18QGKWuXKbUGB5Fw3ImEflW782H85qOBW3uEDDe6DYEM7LwWRSoQI6X9rv_XYeDMMeiYGtdWZvUM31UXYG8q0hzSTellVFUxY4PYCq1BVVKnqtiYv_9cK6bFTvWqY-vMHQn8p9GfP_YELZ2HiY03FV1w5cbNEFrbPCCSYrDn4tSRY20tJJO4R7BoFph5tNz4QYTRBI3N-i48qJz_JzGmbn1Zc1EuZszOPWtw"
	// request.Header.Add("X-Axway-Tenant-Id", "392818510148456")
	// request.Header.Add("Authorization", "Bearer "+token)

	return request, nil
}
