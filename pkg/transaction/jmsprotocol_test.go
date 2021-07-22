package transaction

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJMSProtocolToFromMapString(t *testing.T) {
	jmsProtocol := JMSProtocol{
		Type:             "jms",
		AuthSubjectID:    "authSubjectId",
		JMSMessageID:     "jmsMessageID",
		JMSCorrelationID: "jmsCorrelationID",
		JMSDestination:   "jmsDestination",
		JMSProviderURL:   "jmsProviderURL",
		JMSDeliveryMode:  1,
		JMSPriority:      2,
		JMSReplyTo:       "jmsReplyTo",
		JMSRedelivered:   3,
		JMSTimestamp:     4,
		JMSExpiration:    5,
		JMSType:          "jmsType",
		JMSStatus:        "jmsStatus",
		JMSStatusText:    "jmsStatusText",
	}

	mapString, err := jmsProtocol.ToMapStringString()
	assert.Nil(t, err)
	assert.Equal(t, jmsProtocol.Type, mapString["type"])
	assert.Equal(t, jmsProtocol.AuthSubjectID, mapString["authSubjectId"])
	assert.Equal(t, jmsProtocol.JMSMessageID, mapString["jmsMessageID"])
	assert.Equal(t, jmsProtocol.JMSCorrelationID, mapString["jmsCorrelationID"])
	assert.Equal(t, jmsProtocol.JMSDestination, mapString["jmsDestination"])
	assert.Equal(t, jmsProtocol.JMSProviderURL, mapString["jmsProviderURL"])
	assert.Equal(t, fmt.Sprint(jmsProtocol.JMSDeliveryMode), mapString["jmsDeliveryMode"])
	assert.Equal(t, fmt.Sprint(jmsProtocol.JMSPriority), mapString["jmsPriority"])
	assert.Equal(t, jmsProtocol.JMSReplyTo, mapString["jmsReplyTo"])
	assert.Equal(t, fmt.Sprint(jmsProtocol.JMSRedelivered), mapString["jmsRedelivered"])
	assert.Equal(t, fmt.Sprint(jmsProtocol.JMSTimestamp), mapString["jmsTimestamp"])
	assert.Equal(t, fmt.Sprint(jmsProtocol.JMSExpiration), mapString["jmsExpiration"])
	assert.Equal(t, jmsProtocol.JMSType, mapString["jmsType"])
	assert.Equal(t, jmsProtocol.JMSStatus, mapString["jmsStatus"])
	assert.Equal(t, jmsProtocol.JMSStatusText, mapString["jmsStatusText"])

	newJMSProtocol := JMSProtocol{
		Type:         "jms1",
		JMSStatus:    "jmsStatus1",
		JMSTimestamp: 6,
	}
	err = newJMSProtocol.FromMapStringString(mapString)
	assert.Nil(t, err)
	assert.Equal(t, "jms1", newJMSProtocol.Type)
	assert.Equal(t, "jmsStatus1", newJMSProtocol.JMSStatus)
	assert.Equal(t, 6, newJMSProtocol.JMSTimestamp, 6)
	assert.Equal(t, jmsProtocol.AuthSubjectID, newJMSProtocol.AuthSubjectID)
	assert.Equal(t, jmsProtocol.JMSMessageID, newJMSProtocol.JMSMessageID)
	assert.Equal(t, jmsProtocol.JMSCorrelationID, newJMSProtocol.JMSCorrelationID)
	assert.Equal(t, jmsProtocol.JMSDestination, newJMSProtocol.JMSDestination)
	assert.Equal(t, jmsProtocol.JMSProviderURL, newJMSProtocol.JMSProviderURL)
	assert.Equal(t, jmsProtocol.JMSDeliveryMode, newJMSProtocol.JMSDeliveryMode)
	assert.Equal(t, jmsProtocol.JMSPriority, newJMSProtocol.JMSPriority)
	assert.Equal(t, jmsProtocol.JMSReplyTo, newJMSProtocol.JMSReplyTo)
	assert.Equal(t, jmsProtocol.JMSRedelivered, newJMSProtocol.JMSRedelivered)
	assert.Equal(t, jmsProtocol.JMSExpiration, newJMSProtocol.JMSExpiration)
	assert.Equal(t, jmsProtocol.JMSType, newJMSProtocol.JMSType)
	assert.Equal(t, jmsProtocol.JMSStatusText, newJMSProtocol.JMSStatusText)
}
