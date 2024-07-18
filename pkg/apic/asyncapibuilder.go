package apic

import (
	"fmt"
	"net/url"

	"github.com/Axway/agent-sdk/pkg/util/exception"
	"github.com/swaggest/go-asyncapi/spec-2.4.0"
)

const (
	componentMessageRefTemplate = "#/components/messages/%s"
)

type asyncAPIChannelOpts func(channel *spec.ChannelItem)

func WithKafkaPublishOperationBinding(useGroupID, useClientID bool) asyncAPIChannelOpts {
	return func(c *spec.ChannelItem) {
		kafkaOp := &spec.KafkaOperation{}
		if useGroupID {
			kafkaOp.GroupID = &spec.KafkaOperationGroupID{
				Schema: map[string]interface{}{
					"type": "string",
				},
			}
		}
		if useClientID {
			kafkaOp.ClientID = &spec.KafkaOperationClientID{
				Schema: map[string]interface{}{
					"type": "string",
				},
			}
		}
		c.Publish = &spec.Operation{
			Bindings: &spec.OperationBindingsObject{Kafka: kafkaOp},
		}
	}
}

func WithKafkaSubscribeOperationBinding(useGroupID, useClientID bool) asyncAPIChannelOpts {
	return func(c *spec.ChannelItem) {
		kafkaOp := &spec.KafkaOperation{}
		if useGroupID {
			kafkaOp.GroupID = &spec.KafkaOperationGroupID{
				Schema: map[string]interface{}{
					"type": "string",
				},
			}
		}
		if useClientID {
			kafkaOp.ClientID = &spec.KafkaOperationClientID{
				Schema: map[string]interface{}{
					"type": "string",
				},
			}
		}
		c.Subscribe = &spec.Operation{
			Bindings: &spec.OperationBindingsObject{Kafka: kafkaOp},
		}
	}
}

// TBD ?
// func WithKafkaPublishBinding() asyncAPIChannelOpts {}

type AsyncAPIServerOpts func(*asyncAPIBuilder, *spec.Server)

func WithSaslPlainSecurity(description string) AsyncAPIServerOpts {
	return func(b *asyncAPIBuilder, s *spec.Server) {
		if s.Security == nil {
			s.Security = make([]map[string][]string, 0)
		}
		s.Security = append(s.Security, map[string][]string{
			"saslPlainCreds": {},
		})
		b.securitySchemas["saslPlainCreds"] = &spec.SecurityScheme{
			SaslSecurityScheme: &spec.SaslSecurityScheme{
				SaslPlainSecurityScheme: &spec.SaslPlainSecurityScheme{
					Description: description,
				},
			},
		}
	}
}

func WithSaslScramSecurity(scramMechanism, description string) AsyncAPIServerOpts {
	return func(b *asyncAPIBuilder, s *spec.Server) {
		if s.Security == nil {
			s.Security = make([]map[string][]string, 0)
		}
		s.Security = append(s.Security, map[string][]string{
			"saslScramCreds": {},
		})
		scramType := spec.SaslScramSecuritySchemeTypeScramSha256
		if scramMechanism == "SCRAM-SHA-512" {
			scramType = spec.SaslScramSecuritySchemeTypeScramSha512
		}
		b.securitySchemas["saslScramCreds"] = &spec.SecurityScheme{
			SaslSecurityScheme: &spec.SaslSecurityScheme{
				SaslScramSecurityScheme: &spec.SaslScramSecurityScheme{
					Type:        scramType,
					Description: description,
				},
			},
		}
	}
}

func WithProtocol(protocol, protocolVersion string) AsyncAPIServerOpts {
	return func(b *asyncAPIBuilder, s *spec.Server) {
		s.Protocol = protocol
		s.ProtocolVersion = protocolVersion
	}
}

type AsyncAPIBuilder interface {
	AddServer(name, description, url string, opts ...AsyncAPIServerOpts) AsyncAPIBuilder
	AddChannel(name, description string, opts ...asyncAPIChannelOpts) AsyncAPIBuilder
	SetPublishMessageRef(channelName, componentMessage string) AsyncAPIBuilder
	SetSubscribeMessageRef(channelName, componentMessage string) AsyncAPIBuilder
	AddComponentMessage(msgName, schemaFormat, contentType string, payload map[string]interface{}) AsyncAPIBuilder
	Build(id, title, description, version string) (AsyncSpecProcessor, error)
}

type asyncAPIBuilder struct {
	servers                    map[string]spec.Server
	channels                   map[string]spec.ChannelItem
	channelPublishMessageRef   map[string]string
	channelSubscribeMessageRef map[string]string
	componentMessages          map[string]spec.MessageEntity
	securitySchemas            map[string]*spec.SecurityScheme
}

func CreateAsyncAPIBuilder() AsyncAPIBuilder {
	return &asyncAPIBuilder{
		servers:                    make(map[string]spec.Server),
		channels:                   make(map[string]spec.ChannelItem),
		channelPublishMessageRef:   make(map[string]string),
		channelSubscribeMessageRef: make(map[string]string),
		componentMessages:          make(map[string]spec.MessageEntity),
		securitySchemas:            make(map[string]*spec.SecurityScheme),
	}
}

func (b *asyncAPIBuilder) AddServer(name, description, url string, opts ...AsyncAPIServerOpts) AsyncAPIBuilder {
	server := spec.Server{
		URL:         url,
		Description: description,
	}
	for _, o := range opts {
		o(b, &server)
	}
	b.servers[name] = server
	return b
}

func (b *asyncAPIBuilder) AddChannel(name, description string, opts ...asyncAPIChannelOpts) AsyncAPIBuilder {
	channel := spec.ChannelItem{
		Description: description,
	}

	for _, o := range opts {
		o(&channel)
	}

	b.channels[name] = channel
	return b
}

func setChannelMessageRef(m map[string]string, channelName, messageSubject string) {
	m[channelName] = messageSubject
}

func (b *asyncAPIBuilder) SetPublishMessageRef(channelName, componentMessage string) AsyncAPIBuilder {
	setChannelMessageRef(b.channelPublishMessageRef, channelName, componentMessage)
	return b
}

func (b *asyncAPIBuilder) SetSubscribeMessageRef(channelName, componentMessage string) AsyncAPIBuilder {
	setChannelMessageRef(b.channelSubscribeMessageRef, channelName, componentMessage)
	return b
}

func (b *asyncAPIBuilder) AddComponentMessage(msgName, schemaFormat, contentType string, payload map[string]interface{}) AsyncAPIBuilder {
	msg := spec.MessageEntity{
		Payload:      payload,
		ContentType:  contentType,
		SchemaFormat: schemaFormat,
	}

	b.componentMessages[msgName] = msg
	return b
}

func (b *asyncAPIBuilder) validateInfo(id, title, description, version string) {
	if id == "" {
		exception.Throw(fmt.Errorf("no identifier defined"))
	}
	u, _ := url.Parse(id)
	if u.Scheme == "" {
		exception.Throw(fmt.Errorf("invalid api id, should be uri format"))
	}

	if title == "" {
		exception.Throw(fmt.Errorf("no title defined"))
	}
	if version == "" {
		exception.Throw(fmt.Errorf("no version defined"))
	}
}

func (b *asyncAPIBuilder) validateServers() {
	if len(b.servers) == 0 {
		exception.Throw(fmt.Errorf("no server defined"))
	}
	for _, s := range b.servers {
		if s.URL == "" {
			exception.Throw(fmt.Errorf("invalid server URL"))
		}
		if s.Protocol == "" {
			exception.Throw(fmt.Errorf("invalid server protocol"))
		}
		if s.ProtocolVersion == "" {
			exception.Throw(fmt.Errorf("invalid server protocol version"))
		}
	}

}
func validateMessageRef(channelOp string, channelMessageRef map[string]string, componentMessages map[string]spec.MessageEntity) {
	for channel, message := range channelMessageRef {
		if message == "" {
			exception.Throw(fmt.Errorf("invalid message reference for %s operation in channel %s", channelOp, channel))
		}
		if _, ok := componentMessages[message]; !ok {
			exception.Throw(fmt.Errorf("invalid message reference %s for %s operation in channel %s", message, channelOp, channel))
		}
	}
}

func (b *asyncAPIBuilder) validateChannels() {
	if len(b.channels) == 0 {
		exception.Throw(fmt.Errorf("no channels defined"))
	}
	validateMessageRef("publish", b.channelPublishMessageRef, b.componentMessages)
	validateMessageRef("subscribe", b.channelSubscribeMessageRef, b.componentMessages)
}

func (b *asyncAPIBuilder) validateComponents() {
	for msgName, msg := range b.componentMessages {
		if msgName == "" {
			exception.Throw(fmt.Errorf("invalid message name"))
		}
		if len(msg.Payload) == 0 {
			exception.Throw(fmt.Errorf("invalid message schema"))
		}
	}
}

func (b *asyncAPIBuilder) validate(id, title, description, version string) (err error) {
	exception.Block{
		Try: func() {
			b.validateInfo(id, title, description, version)
			b.validateServers()
			b.validateChannels()
			b.validateComponents()
		},
		Catch: func(e error) {
			err = e
		},
	}.Do()

	return
}

func (b *asyncAPIBuilder) buildServers(api *spec.AsyncAPI) {
	for name, server := range b.servers {
		api.AddServer(name, server)
	}
}

func (b *asyncAPIBuilder) buildChannels(api *spec.AsyncAPI) {
	for name, publishMsgRef := range b.channelPublishMessageRef {
		ch := b.channels[name]
		ch.Publish.Message = &spec.Message{
			Reference: &spec.Reference{
				Ref: fmt.Sprintf(componentMessageRefTemplate, publishMsgRef),
			},
		}
	}
	for name, subscribeMsgRef := range b.channelSubscribeMessageRef {
		ch := b.channels[name]
		ch.Subscribe.Message = &spec.Message{
			Reference: &spec.Reference{
				Ref: fmt.Sprintf(componentMessageRefTemplate, subscribeMsgRef),
			},
		}
	}
	api.WithChannels(b.channels)
}

func (b *asyncAPIBuilder) buildComponents(api *spec.AsyncAPI) {
	components := spec.Components{}
	components.Messages = make(map[string]spec.Message)
	for msgName, msg := range b.componentMessages {
		components.Messages[msgName] = spec.Message{
			OneOf1: &spec.MessageOneOf1{
				MessageEntity: &msg,
			},
		}
	}
	if len(b.securitySchemas) > 0 {
		components.SecuritySchemes = &spec.ComponentsSecuritySchemes{
			MapOfComponentsSecuritySchemesWDValues: make(map[string]spec.ComponentsSecuritySchemesWD),
		}
		for name, securitySchema := range b.securitySchemas {
			components.SecuritySchemes.MapOfComponentsSecuritySchemesWDValues[name] = spec.ComponentsSecuritySchemesWD{
				SecurityScheme: securitySchema,
			}
		}
	}

	api.WithComponents(components)
}

func (b *asyncAPIBuilder) Build(id, title, description, version string) (AsyncSpecProcessor, error) {
	err := b.validate(id, title, description, version)
	if err != nil {
		return nil, err
	}

	api := &spec.AsyncAPI{
		ID: id,
		Info: spec.Info{
			Title:       title,
			Description: description,
			Version:     version,
		},
	}
	b.buildServers(api)
	b.buildChannels(api)
	b.buildComponents(api)

	raw, err := api.MarshalYAML()
	if err != nil {
		return nil, err
	}

	return &asyncApi{spec: api, raw: raw}, nil
}
