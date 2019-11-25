package apic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	apigw "github.com/aws/aws-sdk-go/service/apigateway"
)

// APIServer -
type APIServer struct {
	Name  string `json:"name"`
	Title string `json:"title"`
	// Tags       map[string]interface{} `json:"tags"`		// todo when server ready for key/val pairs
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Spec       map[string]interface{} `json:"spec"`
}

// Spec -
type Spec struct {
	Description string `json:"description"`
}

// CreateAPIServerBodyForAdd -
func (c *Client) CreateAPIServerBodyForAdd(restAPIID, stageName string, restAPI *apigw.RestApi, exportOut *apigw.GetExportOutput, tags map[string]interface{}) ([]byte, error) {

	apiName := aws.StringValue(restAPI.Name)

	// Set tags as Attributes to retain key value pairs.  Add other pertinent data.
	attributes := make(map[string]interface{})
	for key, val := range tags {
		v := val.(*string)
		attributes[key] = *v
	}
	attributes["apiID"] = restAPIID
	attributes["apiName"] = apiName
	attributes["stageName"] = stageName

	// spec needs to adhere to environment schema
	spec := make(map[string]interface{})
	spec["description"] = "API From AWS APIGateway (RestApiId: " + restAPIID + ", StageName: " + stageName + ")"

	// todo temp until api fixed
	newtags := c.MapToStringArray(tags)

	apiServerService := APIServer{
		Name:       strings.ToLower(apiName), // name needs to be path friendly and follows this regex "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*\
		Title:      fmt.Sprintf("%v (Stage: %v)", apiName, stageName),
		Attributes: attributes,
		Spec:       spec,
		Tags:       newtags,
	}

	return json.Marshal(apiServerService)
}

// AddAPIServer -
func (c *Client) AddAPIServer(apiServerBuffer []byte) (string, error) {
	// Unit testing. For now just dummy up a return
	if isUnitTesting() {
		return "12345678", nil
	}

	url := c.cfg.GetAPIServerServicesURL()
	agentMode := c.cfg.GetAgentMode()
	return c.DeployAPI("POST", apiServerBuffer, agentMode, url)
}
