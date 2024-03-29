package apic

// TODO - this file should be able to be removed once Unified Catalog support has been removed
import (
	"net/http"
	"os"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	uc "github.com/Axway/agent-sdk/pkg/apic/unifiedcatalog/models"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePropertyValues(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	assert.NotNil(t, svcClient)
	assert.NotNil(t, mockHTTPClient)

	testCases := []struct {
		name         string
		wantErr      bool
		responseCode int
	}{
		{
			name:         "UC Sub Bad Response",
			responseCode: http.StatusBadRequest,
			wantErr:      true,
		},
		{
			name:         "UC Sub Success",
			responseCode: http.StatusOK,
			wantErr:      false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			subscription := createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"})
			cs := subscription.(*CentralSubscription)
			cs.apicClient = svcClient

			// setup API responses
			mockHTTPClient.SetResponses([]api.MockResponse{{RespCode: tt.responseCode}})

			values := map[string]interface{}{
				"data1": "value1",
				"data2": "value2",
			}
			if err := subscription.UpdatePropertyValues(values); (err != nil) != tt.wantErr {
				t.Errorf("CentralSubscription.UpdatePropertyValues() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				cs := subscription.(*CentralSubscription)
				assert.NotNil(t, cs)
			}
		})
	}
}

func TestUpdateProperties(t *testing.T) {
	wd, _ := os.Getwd()
	svcClient, mockHTTPClient := GetTestServiceClient()
	assert.NotNil(t, svcClient)
	assert.NotNil(t, mockHTTPClient)

	testCases := []struct {
		name            string
		filenames       []string
		responseCodes   []int
		wantErr         bool
		isAccessRequest bool
	}{
		{
			name:          "UC Sub Bad Response 1",
			filenames:     []string{""},
			responseCodes: []int{http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "UC Sub Bad Response 2",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", ""},
			responseCodes: []int{http.StatusOK, http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "UC Sub Bad Response 3",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", "", ""},
			responseCodes: []int{http.StatusOK, http.StatusOK, http.StatusBadRequest},
			wantErr:       true,
		},
		{
			name:          "UC Sub Success",
			filenames:     []string{wd + "/testdata/catalogitemsubscriptiondefprofile.json", "", ""},
			responseCodes: []int{http.StatusOK, http.StatusOK, http.StatusOK},
			wantErr:       false,
		},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			subscription := createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"})
			cs := subscription.(*CentralSubscription)
			cs.apicClient = svcClient

			// setup API responses
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
				t.Errorf("CentralSubscription.UpdateProperties() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateState(t *testing.T) {
	svcClient, mockHTTPClient := GetTestServiceClient()
	assert.NotNil(t, svcClient)
	assert.NotNil(t, mockHTTPClient)

	testCases := []struct {
		name          string
		responseCodes []int
		wantErr       bool
		state         SubscriptionState
		message       string
	}{
		{
			name:          "UC Sub Failed",
			responseCodes: []int{http.StatusBadRequest},
			state:         SubscriptionFailedToSubscribe,
			message:       "failed",
			wantErr:       true,
		},
		{
			name:          "UC Sub Success",
			responseCodes: []int{http.StatusOK},
			state:         SubscriptionActive,
			message:       "",
			wantErr:       false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			subscription := createSubscription("11111", "APPROVED", "11111", map[string]interface{}{"orgId": "11111"})
			cs := subscription.(*CentralSubscription)
			cs.apicClient = svcClient

			// setup API responses
			mockResponses := make([]api.MockResponse, 0)
			for _, code := range tt.responseCodes {
				mockResponses = append(mockResponses,
					api.MockResponse{
						RespCode: code,
					})
			}
			mockHTTPClient.SetResponses(mockResponses)

			if err := subscription.UpdateState(tt.state, tt.message); (err != nil) != tt.wantErr {
				t.Errorf("CentralSubscription.UpdateProperties() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
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
