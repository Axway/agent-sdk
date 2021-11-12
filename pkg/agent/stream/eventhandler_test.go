package stream

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/api"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

var url = "https://tjohnson.dev.ampc.axwaytest.net/apis"
var tenantID = "426937327920148"

func TestEventHandler(t *testing.T) {
	events := make(chan *proto.Event)
	apis, categories, instances := cache.New(), cache.New(), cache.New()

	client := api.NewClient(nil, "")
	c := NewResourceClient(url, client, &mockTokenGetter{}, tenantID)

	sc := NewEventManager(
		events,
		c,
		NewAPISvcHandler(apis),
		NewCategoryHandler(categories),
		NewInstanceHandler(instances),
	)

	go func() {
		err := sc.Start()
		if err != nil {
			logrus.Errorf("stream cache error: %s", err)
			os.Exit(1)
		}
	}()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	filePath := path.Join(dir, "event.json")
	f, err := os.Open(filePath)
	if err != nil {
		t.Fatal(err)
	}

	data, err := ioutil.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}

	event := &proto.Event{}
	err = json.Unmarshal(data, event)
	if err != nil {
		t.Fatal(err)
	}

	events <- event
	time.Sleep(60 * time.Minute)
}

type mockTokenGetter struct {
}

func (m *mockTokenGetter) GetToken() (string, error) {
	return "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzY3NTk1MDUsImlhdCI6MTYzNjc1NTkwNSwiYXV0aF90aW1lIjoxNjM2NzUxNjYwLCJqdGkiOiI0NjBkMjU0ZC0yOTE1LTRhYzUtYTRmOS1jOTgwYjY5NTgwM2UiLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiJjYmEyNzVmMC01MjZjLTQxZGEtODc3Yy1hMWY1NWJlOGIzYTciLCJzZXNzaW9uX3N0YXRlIjoiMTJmOTQ5NzEtMzhiMC00MTgwLTkzZGMtNDVmY2M0NzY1MzZmIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM2NzU1OTA1MzQyRTEyLCJuYW1lIjoiVHJldm9yIEpvaG5zb24iLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0am9obnNvbkBheHdheS5jb20iLCJnaXZlbl9uYW1lIjoiVHJldm9yIiwiZmFtaWx5X25hbWUiOiJKb2huc29uIiwiZW1haWwiOiJ0am9obnNvbkBheHdheS5jb20ifQ.OsFvBvp4YNiWszGnY34C8dJN7gQE0FQjdYOXv7q57j2QT19kMIjpEHB4OYvkCVYLSfWT0BXFZoKJj1CQUYhpv-72dnBot_kC-3l5TVbMNvAfsfK5s6E_mxgD7lR15qK9spp8kjMRyjwsk0QJ2W1jGg5FbZyidTu0-m-QOFIeJYfZaCMHyTQ17OZxpYZfVaPT7nMwZofaXqzzL3iX9P8xPoYHIX5Z486F-dHPSGv2Kp_Wa7rLM5Z7YTsAZMJB-tvsE9J0FQSv_rsBUVWCcsa69xGwm1cC5z8EiDQDPb_I9CRJXKg-CXFwepoK8H90WCPakB2LoE9MUV-Wab5wf-phLA", nil
}
