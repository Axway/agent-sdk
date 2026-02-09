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
			expectErr: true,
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
				return
			}

			assert.Nil(t, err)
			assert.NotNil(t, data)
			assert.Len(t, data, tc.expectedItems)

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
