package traceability

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/Axway/agent-sdk/pkg/agent"
	"github.com/Axway/agent-sdk/pkg/api"
	"github.com/Axway/agent-sdk/pkg/event"
	"github.com/Axway/agent-sdk/pkg/util/log"
)

const metricFlowValue = "api-central-metric"

func makeFlowEvent(flow string) event.Event {
	fields := event.MapStr{
		"message": `{"event":"test"}`,
	}
	if flow != "" {
		fields["fields"] = map[string]interface{}{
			FlowHeader: flow,
		}
	}
	return event.Event{
		Timestamp: time.Now(),
		Fields:    fields,
	}
}

// TestPublishEventsFlowHeader verifies that publishEvents correctly manages the
// axway-target-flow header in client.headers for metric and transaction batches.
func TestPublishEventsFlowHeader(t *testing.T) {
	cases := map[string]struct {
		preSeedHeaders map[string]string // simulate headers left by a prior batch
		events         []event.Event
		wantFlowHeader string // "" means key must be absent from client.headers
	}{
		"metric event sets flow header": {
			events:         []event.Event{makeFlowEvent(metricFlowValue)},
			wantFlowHeader: metricFlowValue,
		},
		"transaction event does not add flow header": {
			events:         []event.Event{makeFlowEvent("")},
			wantFlowHeader: "",
		},
		"stale metric flow header cleared by subsequent transaction batch": {
			preSeedHeaders: map[string]string{FlowHeader: metricFlowValue},
			events:         []event.Event{makeFlowEvent("")},
			wantFlowHeader: "",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			agent.InitializeForTest(nil)

			headers := make(map[string]string)
			for k, v := range tc.preSeedHeaders {
				headers[k] = v
			}

			client := &HTTPClient{
				Connection: Connection{
					connected: true,
					encoder:   newJSONEncoder(nil),
					api:       &api.MockHTTPClient{ResponseCode: http.StatusOK},
				},
				headers: headers,
				logger:  log.NewFieldLogger(),
			}

			client.publishEvents(tc.events) //nolint:errcheck

			if tc.wantFlowHeader == "" {
				assert.NotContains(t, client.headers, FlowHeader)
			} else {
				assert.Equal(t, tc.wantFlowHeader, client.headers[FlowHeader])
			}
		})
	}
}
