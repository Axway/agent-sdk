package apic

import (
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
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

	testCases := []struct {
		name          string
		filenames     []string
		responseCodes []int
		wantErr       bool
	}{
		{
			name:          "Bad Response 1",
			filenames:     []string{""},
			responseCodes: []int{http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "Bad Response 2",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", ""},
			responseCodes: []int{http.StatusOK, http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "Bad Response 3",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", "", ""},
			responseCodes: []int{http.StatusOK, http.StatusOK, http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "Success",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", "", ""},
			responseCodes: []int{http.StatusOK, http.StatusOK, http.StatusOK},
			wantErr:       false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			mockResponses := make([]api.MockResponse, 0)
			for i, code := range tt.responseCodes {
				mockResponses = append(mockResponses,
					api.MockResponse{
						FileName: tt.filenames[i],
						RespCode: code,
					})
			}
			mockHTTPClient.SetResponses(mockResponses)
			if err := subscription.UpdateProperties("11111"); (err != nil) != tt.wantErr {
				t.Errorf("CentralSubscription.UpdatePropertyValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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

	accReq := createAccessRequestSubscription("access-request", string(AccessRequestProvisioning), "service-instance", map[string]interface{}{"orgId": "33333"})
	assert.Equal(t, "bbunny", accReq.GetCreatedUserID())
	assert.Equal(t, "access-request", accReq.GetID())
	assert.Equal(t, "access-request", accReq.GetName())
	assert.Equal(t, "service-instance", accReq.GetApicID())
	assert.Equal(t, "2222", accReq.GetRemoteAPIID())
	assert.Equal(t, "stage", accReq.GetRemoteAPIStage())
	assert.Equal(t, "access-request", accReq.GetCatalogItemID())
	assert.Equal(t, AccessRequestProvisioning, accReq.GetState())
	assert.Equal(t, "33333", accReq.GetPropertyValue("orgId"))
	assert.Equal(t, "", accReq.GetPropertyValue("foo"))
}

func TestCentralSubscription_UpdatePropertyValues(t *testing.T) {
	type fields struct {
		Subscription            Subscription
		CatalogItemSubscription *uc.CatalogItemSubscription
		ApicID                  string
		RemoteAPIID             string
		RemoteAPIStage          string
		apicClient              *ServiceClient
		RemoteAPIAttributes     map[string]string
	}
	type args struct {
		values map[string]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &CentralSubscription{
				Subscription:            tt.fields.Subscription,
				CatalogItemSubscription: tt.fields.CatalogItemSubscription,
				ApicID:                  tt.fields.ApicID,
				RemoteAPIID:             tt.fields.RemoteAPIID,
				RemoteAPIStage:          tt.fields.RemoteAPIStage,
				apicClient:              tt.fields.apicClient,
				RemoteAPIAttributes:     tt.fields.RemoteAPIAttributes,
			}
			if err := s.UpdatePropertyValues(tt.args.values); (err != nil) != tt.wantErr {
				t.Errorf("CentralSubscription.UpdatePropertyValues() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
