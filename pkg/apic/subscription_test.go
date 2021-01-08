package apic

import (
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestUpdateProperties(t *testing.T) {
	wd, _ := os.Getwd()
	svcClient, mockHTTPClient := GetTestServiceClient()
	assert.NotNil(t, svcClient)
	assert.NotNil(t, mockHTTPClient)

	subscription := createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"})
	cs := subscription.(*CentralSubscription)
	cs.apicClient = svcClient

	// fail
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusBadRequest,
		},
	})
	err := subscription.UpdateProperties("11111")
	assert.NotNil(t, err)

	// fail
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile.json",
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusBadRequest, // update status
		},
	})
	err = subscription.UpdateProperties("11111")
	assert.NotNil(t, err)

	// failure
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile.json",
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusOK, // update status
		},
		{
			RespCode: http.StatusBadRequest, // update property
		},
	})
	err = subscription.UpdateProperties("11111")
	assert.NotNil(t, err)

	// success
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			FileName: wd + "/testdata/catalogitemsubscriptiondefprofile.json",
			RespCode: http.StatusOK,
		},
		{
			RespCode: http.StatusOK, // update status
		},
		{
			RespCode: http.StatusOK, // update property
		},
	})
	err = subscription.UpdateProperties("11111")
	assert.Nil(t, err)

}
