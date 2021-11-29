package stream

import (
	"fmt"
	"testing"

	"github.com/Axway/agent-sdk/pkg/util/healthcheck"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/cache"
)

var host = "https://tjohnson.dev.ampc.axwaytest.net"
var id = "426937327920148"
var topic = "/management/v1alpha1/watchtopics/mock-watch-topic"
var tk = "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzcwMDc2NDgsImlhdCI6MTYzNzAwNDA0OCwiYXV0aF90aW1lIjoxNjM2OTkxMTQ4LCJqdGkiOiJkYmI0MGU1Ny03YzIxLTRjZGYtODk1Yy1jMmFjMDJkOWUyNjciLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiJkYzE1MjdmMy0zNDBiLTQzNjEtYjk4OC1hMzNjMTA5OTE4ZGQiLCJzZXNzaW9uX3N0YXRlIjoiOWNmODk5N2UtZTMwOS00YzhiLWFiMTQtNDk4OWE0Mzc3MTgyIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM3MDA0MDQ4MDQ3RTEyLCJuYW1lIjoiVHJldm9yIEpvaG5zb24iLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0am9obnNvbkBheHdheS5jb20iLCJnaXZlbl9uYW1lIjoiVHJldm9yIiwiZmFtaWx5X25hbWUiOiJKb2huc29uIiwiZW1haWwiOiJ0am9obnNvbkBheHdheS5jb20ifQ.VbaCMYITBfHQl_msDGB6XwZ62aH26Mauwib2OUMyKHN8zBRNJTV-qyvznIDt6roStLdneP2JVHXMDPk6sAd4Qy5bnfkqXg1vnLC1ZoqCxG9-frQT7Ve8pOGxFPmg2ZyJ8XM4VTbnj1fLAfkiziWf_fber-46g0zryYt6RBpnZsZ3dKjCInkmRTmvrriElcvCyJpLrJ5g-J50v99GItevjuUABXBYd7chdyRqkymBhwSJLUexiPNc9vrj__kI8DkEloed0f0QoBu4_M0uJr-28CjigEHF0cqdyuj-dkzCiytJeQWa_N-gxqcRh3TYZLe6AK2sAqqb76PR84jkE2-zQw"

// Should call newStreamService without error
func TestClient_newStreamService(t *testing.T) {
	c := NewClient(
		host,
		id,
		topic,
		&mockTokenGetter{
			token: tk,
		},
		&api.MockHTTPClient{},
		&mockManager{},
		NewAPISvcHandler(cache.New()),
		NewInstanceHandler(cache.New()),
		NewCategoryHandler(cache.New()),
	)
	c.newEventManager = mockNewEventManager
	err := c.newStreamService()
	assert.Nil(t, err)
}

// Should call newStreamService and handle an error
func TestClient_Start(t *testing.T) {
	c := NewClient(
		host,
		id,
		topic,
		&mockTokenGetter{
			token: tk,
		},
		&api.MockHTTPClient{},
		&mockManager{
			err: fmt.Errorf("err"),
		},
		NewAPISvcHandler(cache.New()),
		NewInstanceHandler(cache.New()),
		NewCategoryHandler(cache.New()),
	)
	c.newEventManager = mockNewEventManager
	err := c.Start()
	assert.NotNil(t, err)

	hc := c.HealthCheck()("")

	assert.Equal(t, healthcheck.OK, hc.Result)
}

type mockManager struct {
	err error
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
	return true
}

type mockEventManager struct{}

func (m mockEventManager) Listen() error {
	return nil
}

func mockNewEventManager(_ chan *proto.Event, _ resourceGetter, _ ...Handler) EventListener {
	return &mockEventManager{}
}
