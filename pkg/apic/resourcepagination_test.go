package apic

import (
	"net/http"
	"testing"

	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestGetAPIV1ResourceInstancesWithPageSize(t *testing.T) {
	const url = "/test"

	testCases := map[string]struct {
		skip             bool
		startPageSize    int
		responses        []api.MockResponse
		secondCall       bool // call a second time after error hit for retries
		expectErr        bool
		expectedItems    int
		expectedPageSize int
	}{
		"no error when no response": {
			responses:        []api.MockResponse{},
			expectedPageSize: -1,
		},
		"pageSize is halved and saved after context error": {
			startPageSize: 100,
			responses: []api.MockResponse{
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode: http.StatusOK,
					RespData: "[]",
				},
			},
			expectedPageSize: 50,
		},
		"pageSize is halved twice, but kept an its minimum of": {
			startPageSize: minPageSize * 2,
			responses: []api.MockResponse{
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode: http.StatusOK,
					RespData: "[]",
				},
			},
			expectedPageSize: minPageSize,
		},
		"expect err after retries exhausted": {
			startPageSize: minPageSize * 2,
			responses: []api.MockResponse{
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode: http.StatusOK,
					RespData: "[]",
				},
			},
			expectErr:        true,
			expectedPageSize: 5,
		},
		"expect page size to half further after initial retry exhaustion to minimum": {
			startPageSize: 50,
			responses: []api.MockResponse{
				{ //50
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //25
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //12
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //6 - max retries
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //3 - 5
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //2 - 5
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{ //1 - 5
					RespCode:  http.StatusRequestTimeout,
					ErrString: "context deadline exceeded",
				},
				{
					RespCode: http.StatusOK,
					RespData: "[]",
				},
			},
			secondCall:       true,
			expectErr:        true,
			expectedPageSize: 5,
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			if tc.skip {
				return
			}
			client, httpClient := GetTestServiceClient()
			httpClient.SetResponses(tc.responses)

			data, err := client.GetAPIV1ResourceInstancesWithPageSize(map[string]string{"key": "value"}, url, tc.startPageSize)
			if tc.expectErr {
				assert.Nil(t, data)
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, data)
				assert.Len(t, data, tc.expectedItems)
			}

			if tc.secondCall {
				data, err := client.GetAPIV1ResourceInstancesWithPageSize(map[string]string{"key": "value"}, url, tc.startPageSize)
				assert.Nil(t, err)
				assert.NotNil(t, data)
				assert.Len(t, data, tc.expectedItems)
			}

			size, ok := client.getPageSize(client.createAPIServerURL(url))
			if tc.expectedPageSize >= 0 {
				assert.True(t, ok)
				assert.Equal(t, tc.expectedPageSize, size)
				return
			}
			assert.False(t, ok)
		})
	}
}
