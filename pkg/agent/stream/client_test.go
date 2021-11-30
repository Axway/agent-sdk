package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util/healthcheck"
	hc "github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/cache"
)

var host = "https://tjohnson.dev.ampc.axwaytest.net"
var id = "426937327920148"
var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"
var tk = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzcwMDc2NDgsImlhdCI6MTYzNzAwNDA0OCwiYXV0aF90aW1lIjoxNjM2OTkxMTQ4LCJqdGkiOiJkYmI0MGU1Ny03YzIxLTRjZGYtODk1Yy1jMmFjMDJkOWUyNjciLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiJkYzE1MjdmMy0zNDBiLTQzNjEtYjk4OC1hMzNjMTA5OTE4ZGQiLCJzZXNzaW9uX3N0YXRlIjoiOWNmODk5N2UtZTMwOS00YzhiLWFiMTQtNDk4OWE0Mzc3MTgyIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM3MDA0MDQ4MDQ3RTEyLCJuYW1lIjoiVHJldm9yIEpvaG5zb24iLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0am9obnNvbkBheHdheS5jb20iLCJnaXZlbl9uYW1lIjoiVHJldm9yIiwiZmFtaWx5X25hbWUiOiJKb2huc29uIiwiZW1haWwiOiJ0am9obnNvbkBheHdheS5jb20ifQ.VbaCMYITBfHQl_msDGB6XwZ62aH26Mauwib2OUMyKHN8zBRNJTV-qyvznIDt6roStLdneP2JVHXMDPk6sAd4Qy5bnfkqXg1vnLC1ZoqCxG9-frQT7Ve8pOGxFPmg2ZyJ8XM4VTbnj1fLAfkiziWf_fber-46g0zryYt6RBpnZsZ3dKjCInkmRTmvrriElcvCyJpLrJ5g-J50v99GItevjuUABXBYd7chdyRqkymBhwSJLUexiPNc9vrj__kI8DkEloed0f0QoBu4_M0uJr-28CjigEHF0cqdyuj-dkzCiytJeQWa_N-gxqcRh3TYZLe6AK2sAqqb76PR84jkE2-zQw"

func TestClient(t *testing.T) {
	tests := []struct {
		name   string
		status bool
		err    error
		hasErr bool
	}{
		{
			name:   "should return an OK status on the healthcheck",
			status: true,
			err:    nil,
			hasErr: false,
		},
		{
			name:   "should return a FAIL status on the healthcheck",
			status: false,
			err:    nil,
			hasErr: false,
		},
		{
			name:   "should handle an error from the manager",
			status: true,
			err:    fmt.Errorf("error"),
			hasErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := NewClient(
				host,
				id,
				topic,
				&mockTokenGetter{
					token: tk,
				},
				&api.MockHTTPClient{},
				&mockManager{
					err:    tc.err,
					status: tc.status,
				},
				NewAPISvcHandler(cache.New()),
				NewInstanceHandler(cache.New()),
				NewCategoryHandler(cache.New()),
			)
			c.newEventManager = mockNewEventManager
			err := c.Start()
			if tc.hasErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			status := c.HealthCheck()("")

			if tc.status == true {
				assert.Equal(t, healthcheck.OK, status.Result)

			} else {
				assert.Equal(t, healthcheck.FAIL, status.Result)
			}
		})
	}
}

func TestRestart(t *testing.T) {
	f := func(s hc.StatusLevel) hc.CheckStatus {
		return func(string) *hc.Status {
			return &hc.Status{
				Result: s,
			}
		}
	}

	status := Restart(f(hc.OK), mockStarter{err: nil})("")
	assert.Equal(t, hc.OK, status.Result)

	status = Restart(f(hc.FAIL), mockStarter{err: fmt.Errorf("fail")})("")
	assert.Equal(t, hc.FAIL, status.Result)
}

type mockStarter struct {
	err error
}

func (m mockStarter) Start() error {
	return m.err
}

type mockManager struct {
	err    error
	status bool
}

func (m mockManager) RegisterWatch(_ string, _ chan *proto.Event, _ chan error) (string, error) {
	return "", m.err
}

func (m mockManager) CloseWatch(_ string) error {
	return nil
}

func (m mockManager) Close() {
}

func (m mockManager) Status() bool {
	return m.status
}

type mockEventManager struct{}

func (m mockEventManager) Listen() error {
	return nil
}

func mockNewEventManager(_ chan *proto.Event, _ resourceGetter, _ ...Handler) EventListener {
	return &mockEventManager{}
}
