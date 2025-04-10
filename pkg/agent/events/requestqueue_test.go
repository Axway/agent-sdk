package events

import (
	"sync"
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/watchmanager/proto"
	"github.com/stretchr/testify/assert"
)

func TestRequestQueue(t *testing.T) {
	cases := []struct {
		name           string
		writeErr       bool
		queueActivated bool
	}{
		{
			name:           "write error when queue is not active",
			writeErr:       true,
			queueActivated: false,
		},
		{
			name:           "write success with active queue",
			queueActivated: true,
		},
	}
	for _, tc := range cases {
		requestCh := make(chan *proto.Request, 1)
		t.Run(tc.name, func(t *testing.T) {
			q := NewRequestQueue(requestCh)
			var receivedReq *proto.Request
			wg := sync.WaitGroup{}
			if tc.queueActivated {
				wg.Add(1)
				q.Start()
				time.Sleep(time.Second)
				go func() {
					receivedReq = <-requestCh
					q.Stop()
					wg.Done()
				}()
			}

			req := &proto.Request{
				RequestType: proto.RequestType_AGENT_STATUS.Enum(),
				AgentStatus: &proto.AgentStatus{
					State: "running",
				},
			}
			writeErr := q.Write(req)
			if tc.writeErr {
				assert.NotNil(t, writeErr)
				return
			}
			assert.Nil(t, writeErr)

			wg.Wait()
			assert.Equal(t, req, receivedReq)
			time.Sleep(time.Millisecond * 500)
			assert.False(t, q.IsActive())
		})
	}

}
