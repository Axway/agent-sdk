package transaction

// JMSProtocolBuilder - Interface to build the JMS protocol details for transaction log event
type JMSProtocolBuilder interface {
	SetMessageID(messageID string) JMSProtocolBuilder
	SetCorrelationID(correlationID string) JMSProtocolBuilder
	SetAuthSubjectID(authSubjectID string) JMSProtocolBuilder
	SetDestination(destination string) JMSProtocolBuilder
	SetProviderURL(providerURL string) JMSProtocolBuilder
	SetDeliveryMode(deliveryMode int) JMSProtocolBuilder
	SetPriority(priority int) JMSProtocolBuilder
	SetReplyTo(replyTo string) JMSProtocolBuilder
	SetRedelivered(redelivered int) JMSProtocolBuilder
	SetTimestamp(timestamp int) JMSProtocolBuilder
	SetExpiration(expiration int) JMSProtocolBuilder
	SetJMSType(jmsType string) JMSProtocolBuilder
	SetStatus(status string) JMSProtocolBuilder
	SetStatusText(statusText string) JMSProtocolBuilder

	Build() (TransportProtocol, error)
}

type jmsProtocolBuilder struct {
	JMSProtocolBuilder
	err         error
	jmsProtocol *JMSProtocol
}

// NewJMSProtocolBuilder - Creates a new JMS protocol builder
func NewJMSProtocolBuilder() JMSProtocolBuilder {
	builder := &jmsProtocolBuilder{
		jmsProtocol: &JMSProtocol{
			Type: "jms",
		},
	}
	return builder
}

func (b *jmsProtocolBuilder) SetMessageID(messageID string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSMessageID = messageID
	return b
}

func (b *jmsProtocolBuilder) SetCorrelationID(correlationID string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSCorrelationID = correlationID
	return b
}

func (b *jmsProtocolBuilder) SetAuthSubjectID(authSubjectID string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.AuthSubjectID = authSubjectID
	return b
}

func (b *jmsProtocolBuilder) SetDestination(destination string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSDestination = destination
	return b
}

func (b *jmsProtocolBuilder) SetProviderURL(providerURL string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSProviderURL = providerURL
	return b
}

func (b *jmsProtocolBuilder) SetDeliveryMode(deliveryMode int) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSDeliveryMode = deliveryMode
	return b
}

func (b *jmsProtocolBuilder) SetPriority(priority int) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSPriority = priority
	return b
}

func (b *jmsProtocolBuilder) SetReplyTo(replyTo string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSReplyTo = replyTo
	return b
}

func (b *jmsProtocolBuilder) SetRedelivered(redelivered int) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSRedelivered = redelivered
	return b
}

func (b *jmsProtocolBuilder) SetTimestamp(timestamp int) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSTimestamp = timestamp
	return b
}

func (b *jmsProtocolBuilder) SetExpiration(expiration int) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSExpiration = expiration
	return b
}

func (b *jmsProtocolBuilder) SetJMSType(jmsType string) JMSProtocolBuilder {
	b.jmsProtocol.JMSType = jmsType
	return b
}

func (b *jmsProtocolBuilder) SetStatus(status string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSStatus = status
	return b
}

func (b *jmsProtocolBuilder) SetStatusText(statusText string) JMSProtocolBuilder {
	if b.err != nil {
		return b
	}
	b.jmsProtocol.JMSStatusText = statusText
	return b
}

func (b *jmsProtocolBuilder) Build() (TransportProtocol, error) {
	if b.err != nil {
		return nil, b.err
	}
	return b.jmsProtocol, nil
}
