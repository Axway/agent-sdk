package stream

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/Axway/agent-sdk/pkg/cache"
	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
)

var url = "https://tjohnson.dev.ampc.axwaytest.net/apis"
var tenantID = "426937327920148"

func TestEventHandler(t *testing.T) {
	events := make(chan *proto.Event)
	apis, categories := cache.New(), cache.New()

	c := NewResourceInstanceClient(url, &http.Client{}, &mockTokenGetter{}, tenantID)

	sc := NewEventManager(events, c, apis, categories)

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
	return "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzY1Nzg1NzcsImlhdCI6MTYzNjU3NDk3NywiYXV0aF90aW1lIjoxNjM2NTcwMDEwLCJqdGkiOiI0NzhlNGVkZC1kZWRiLTQxM2YtYjgxMi04NGNhMmM0ZDk0ZDMiLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiJjZDRiYzJiNi1iZTE3LTQ2NTAtODZmYy1jYzc3M2IzN2NiZGIiLCJzZXNzaW9uX3N0YXRlIjoiNzBjMzRiNDctNDM4NS00NGEyLWFjMDMtODk5ZDI5NGMxZTRmIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM2NTc0OTc3MDM4RTEyLCJuYW1lIjoiVHJldm9yIEpvaG5zb24iLCJwcmVmZXJyZWRfdXNlcm5hbWUiOiJ0am9obnNvbkBheHdheS5jb20iLCJnaXZlbl9uYW1lIjoiVHJldm9yIiwiZmFtaWx5X25hbWUiOiJKb2huc29uIiwiZW1haWwiOiJ0am9obnNvbkBheHdheS5jb20ifQ.LKovCWfsj5R0MK5eI-auDCOML05_gRxqq0yW-JvqKMOzHvha5PLMX6yd4aEkXg10bXLwLYdrrp7nsJAAIcZRdc7Wz4-Ef2e7tmX8Hk_2xtVIuXNtulqOvmoe89NOBZQ0s4e3B7yzPJ_HPGVAIzq_0-q78oQhlPlij-K_YiUn2zCzuGYxKzBSpfooiF-zTKW0TySwWXcPlQt7YJS7txKOX4qfV4p7GSZlYnZziOCPiPzW34WvB5__hA2MYziKf2bHXA7nRJOH8KQlH0cJSeG4bbDM14pcuzgheI9gLWgighLi95oEoAOXYcnOyyiagSIqglj-SaPawmfc9Z6B7OQVAA", nil
}
