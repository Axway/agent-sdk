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

	sc := NewEventManager(events, c, apis, categories, instances)

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
	return "Bearer eyJhbGciOiJSUzI1NiIsInR5cCIgOiAiSldUIiwia2lkIiA6ICJUNXJfaUwwbWJXUWpFQS1JcWNDSkFKaXlia0k4V2xrUnd0YVFQV0ZlWjJJIn0.eyJleHAiOjE2MzY3NTcxNzUsImlhdCI6MTYzNjc1MzU3NSwiYXV0aF90aW1lIjoxNjM2NzUxNjYwLCJqdGkiOiI2YTU2ZmVjNC1kOTQwLTQzZjItOTBmNS0wN2M4N2ExYzY3MGQiLCJpc3MiOiJodHRwczovL2xvZ2luLXByZXByb2QuYXh3YXkuY29tL2F1dGgvcmVhbG1zL0Jyb2tlciIsImF1ZCI6WyJicm9rZXIiLCJhY2NvdW50IiwiYXBpY2VudHJhbCJdLCJzdWIiOiIzMWM5NTM4Yy0zMmFkLTRkYjctYTVlNC0wMjA0NzBhY2NkMGMiLCJ0eXAiOiJCZWFyZXIiLCJhenAiOiJhcGljZW50cmFsIiwibm9uY2UiOiI4YTNmYTMwOS00Zjg5LTQ4NGMtODNkMC1jMDdhYmY2ZTNmMjYiLCJzZXNzaW9uX3N0YXRlIjoiMTJmOTQ5NzEtMzhiMC00MTgwLTkzZGMtNDVmY2M0NzY1MzZmIiwiYWNyIjoiMCIsInJlYWxtX2FjY2VzcyI6eyJyb2xlcyI6WyJhZG1pbmlzdHJhdG9yIiwib2ZmbGluZV9hY2Nlc3MiLCJ1bWFfYXV0aG9yaXphdGlvbiIsImF4d2F5X2VtcGxveWVlIl19LCJyZXNvdXJjZV9hY2Nlc3MiOnsiYnJva2VyIjp7InJvbGVzIjpbInJlYWQtdG9rZW4iXX0sImFjY291bnQiOnsicm9sZXMiOlsibWFuYWdlLWFjY291bnQiLCJtYW5hZ2UtYWNjb3VudC1saW5rcyIsInZpZXctcHJvZmlsZSJdfX0sInNjb3BlIjoib3BlbmlkIiwic3ViIjoiMzFjOTUzOGMtMzJhZC00ZGI3LWE1ZTQtMDIwNDcwYWNjZDBjIiwiaWRlbnRpdHlfcHJvdmlkZXIiOiJhenVyZS1hZCIsInVwZGF0ZWRfYXQiOjEuNjM2NzUzNTc1NzhFMTIsIm5hbWUiOiJUcmV2b3IgSm9obnNvbiIsInByZWZlcnJlZF91c2VybmFtZSI6InRqb2huc29uQGF4d2F5LmNvbSIsImdpdmVuX25hbWUiOiJUcmV2b3IiLCJmYW1pbHlfbmFtZSI6IkpvaG5zb24iLCJlbWFpbCI6InRqb2huc29uQGF4d2F5LmNvbSJ9.TxlTpEwQqYESbyujrqHn6a8D16tH8dAUd4k-1xn9KU09mTPwnPlPLQVk8Xg92d6g6NSivW1vGvR8v-9i285m_u1VVSnHp8j7mHKW9oLyu7khArSOWcdyYHK-7xqnE4jlX1ybCsGdpj_OfbaT_mi_ExphWeb4HNNJB0FXZHacSOV2ULZzdBRd-L1MfIcAEaRxCL5UK3NZdkR0zVrBBzAppYbnhmy4rY4g895fT1MbfcKgq-hC_S3FZIh2uT3ygaYEyIMbGYHARX79jYpRdNbMwb8Nj4TvxJl6w3elOsqcDdM5dD_UAG5cTnAOByFO8qnDNjnrYxkGlgwkUogYLFxzYg", nil
}
