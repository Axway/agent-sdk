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

func TestUpdateState(t *testing.T) {
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
	err := subscription.UpdateState("FAILED", "failure")
	assert.NotNil(t, err)

	// success
	mockHTTPClient.SetResponses([]api.MockResponse{
		{
			RespCode: http.StatusOK, // update state
		},
	})
	err = subscription.UpdateState("FAILED", "failure")
	assert.Nil(t, err)
}

func TestGetters(t *testing.T) {
	subscription := createSubscription("11111", "APPROVED", "22222", map[string]interface{}{"orgId": "33333"})
	assert.Equal(t, "bbunny", subscription.GetCreatedUserID())
	assert.Equal(t, "11111", subscription.GetID())
	assert.Equal(t, "testsubscription", subscription.GetName())
	assert.Equal(t, "1111", subscription.GetApicID())
	assert.Equal(t, "2222", subscription.GetRemoteAPIID())
	assert.Equal(t, "stage", subscription.GetRemoteAPIStage())
	assert.Equal(t, "22222", subscription.GetCatalogItemID())
	assert.Equal(t, SubscriptionApproved, subscription.GetState())
	assert.Equal(t, "33333", subscription.GetPropertyValue("orgId"))
	assert.Equal(t, "", subscription.GetPropertyValue("foo"))
}
