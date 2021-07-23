package transaction

import (
	"testing"
	"time"

	"github.com/Axway/agent-sdk/pkg/traceability/redaction"
	"github.com/stretchr/testify/assert"
)

func createJMSProtocol(msgID, correlationID, jmsType, url, destination, replyTo, status string, mode, priority, exp, timestamp int) (TransportProtocol, error) {
	redaction.SetupGlobalRedaction(redaction.Config{})
	return NewJMSProtocolBuilder().
		SetMessageID(msgID).
		SetCorrelationID(correlationID).
		SetAuthSubjectID("authSubject").
		SetDestination(destination).
		SetProviderURL(url).
		SetDeliveryMode(mode).
		SetPriority(priority).
		SetReplyTo(replyTo).
		SetRedelivered(0).
		SetTimestamp(timestamp).
		SetExpiration(exp).
		SetJMSType(jmsType).
		SetStatus(status).
		SetStatusText("OK").
		Build()
}
func TestJMSProtocolBuilder(t *testing.T) {
	timeStamp := int(time.Now().Unix())
	jmsProtocol, err := createJMSProtocol("m1", "c1", "jms", "jms://test", "dest", "source", "Success", 1, 1, 2, timeStamp)
	assert.Nil(t, err)
	assert.NotNil(t, jmsProtocol)
}
