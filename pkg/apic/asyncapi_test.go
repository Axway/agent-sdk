package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type asyncServer struct {
	name            string
	description     string
	url             string
	protocol        string
	protocolVersion string
	useSaslScram    bool
	useSaslPlain    bool
}

type asyncChannel struct {
	name             string
	publish          bool
	subscribe        bool
	publishMessage   string
	subscribeMessage string
}

type asyncMessage struct {
	name        string
	format      string
	contentType string
	payload     map[string]interface{}
}

func TestAsyncAPIGenerator(t *testing.T) {
	tests := map[string]struct {
		expectBuilder bool
		id            string
		title         string
		version       string
		servers       []asyncServer
		messages      []asyncMessage
		channels      []asyncChannel
	}{
		"no id in info": {
			expectBuilder: true,
		},
		"invalid id in info": {
			id:            "test",
			expectBuilder: true,
		},
		"no title in info": {
			id:            "kafka://test",
			expectBuilder: true,
		},
		"no version in info": {
			id:            "kafka://test",
			title:         "test",
			expectBuilder: true,
		},
		"no servers": {
			id:            "kafka://test",
			title:         "test",
			version:       "1.0.0",
			expectBuilder: true,
		},
		"servers with no url": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:        "test",
					description: "test",
					url:         "",
				},
			},
			expectBuilder: true,
		},
		"servers with no protocol": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:        "test",
					description: "test",
					url:         "PLAINTEXT://localhost",
				},
			},
			expectBuilder: true,
		},
		"servers with no protocol version": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:        "test",
					description: "test",
					url:         "PLAINTEXT://localhost",
					protocol:    "kafka",
				},
			},
			expectBuilder: true,
		},
		"no channels": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			expectBuilder: true,
		},
		"servers with invalid component message message ref": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			channels: []asyncChannel{
				{
					name:           "test",
					publish:        true,
					subscribe:      true,
					publishMessage: "",
				},
			},
			expectBuilder: true,
		},
		"servers with no component message for publish message ref": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			channels: []asyncChannel{
				{
					name:             "test",
					publish:          true,
					subscribe:        true,
					publishMessage:   "order",
					subscribeMessage: "order",
				},
			},
			expectBuilder: true,
		},
		"servers with no component message for subscribe message ref": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			channels: []asyncChannel{
				{
					name:             "test",
					publish:          true,
					subscribe:        true,
					publishMessage:   "order-pub",
					subscribeMessage: "order-sub",
				},
			},
			messages: []asyncMessage{
				{
					name:    "order-pub",
					format:  "json",
					payload: map[string]interface{}{"schema": map[string]interface{}{}},
				},
			},
			expectBuilder: true,
		},
		"invalid component message name": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			channels: []asyncChannel{
				{
					name:             "test",
					publish:          true,
					subscribe:        true,
					publishMessage:   "order-pub",
					subscribeMessage: "order-sub",
				},
			},
			messages: []asyncMessage{
				{
					name:   "order-pub",
					format: "json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
				{
					name:   "order-sub",
					format: "json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
				{
					name:   "",
					format: "json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
			},
			expectBuilder: true,
		},
		"invalid component message payload": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test",
					description:     "test",
					url:             "PLAINTEXT://localhost",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
				},
			},
			channels: []asyncChannel{
				{
					name:             "test",
					publish:          true,
					subscribe:        true,
					publishMessage:   "order-pub",
					subscribeMessage: "order-sub",
				},
			},
			messages: []asyncMessage{
				{
					name:   "order-pub",
					format: "json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
				{
					name:   "order-sub",
					format: "json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
				{
					name:   "test",
					format: "json",
				},
			},
			expectBuilder: true,
		},
		"valid async api spec": {
			id:      "kafka://test",
			title:   "test",
			version: "1.0.0",
			servers: []asyncServer{
				{
					name:            "test-plain",
					description:     "test",
					url:             "SASL_SSL://localhost:9092",
					protocol:        "kafka",
					protocolVersion: "1.0.0",
					useSaslScram:    false,
					useSaslPlain:    true,
				},
			},
			channels: []asyncChannel{
				{
					name:             "test",
					publish:          true,
					subscribe:        true,
					publishMessage:   "order",
					subscribeMessage: "order",
				},
			},
			messages: []asyncMessage{
				{
					name:        "order",
					format:      "application/vnd.oai.openapi+json;version=3.0.0",
					contentType: "application/json",
					payload: map[string]interface{}{
						"schema": map[string]interface{}{},
					},
				},
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			builder := CreateAsyncAPIBuilder()
			for _, serverDetail := range tt.servers {
				opts := make([]AsyncAPIServerOpts, 0)
				if serverDetail.useSaslPlain {
					opts = append(opts, WithSaslPlainSecurity("SASL_PLAIN"))
				}
				if serverDetail.useSaslScram {
					opts = append(opts, WithSaslScramSecurity("SCRAM-SHA-512", "SASL_PLAINTEXT"))
				}
				opts = append(opts, WithProtocol(serverDetail.protocol, serverDetail.protocolVersion))
				builder.AddServer(serverDetail.name, serverDetail.description, serverDetail.url, opts...)
			}

			for _, msg := range tt.messages {
				builder.AddComponentMessage(msg.name, msg.format, msg.contentType, msg.payload)
			}
			for _, channel := range tt.channels {
				opts := make([]asyncAPIChannelOpts, 0)
				if channel.publish {
					opts = append(opts, WithKafkaPublishOperationBinding(true, true))
				}
				if channel.subscribe {
					opts = append(opts, WithKafkaSubscribeOperationBinding(true, true))
				}
				builder.AddChannel(channel.name, channel.name, opts...)
				builder.SetPublishMessageRef(channel.name, channel.publishMessage)
				builder.SetSubscribeMessageRef(channel.name, channel.subscribeMessage)
			}

			spec, err := builder.Build(tt.id, tt.title, tt.title, tt.version)
			if tt.expectBuilder {
				assert.NotNil(t, err)
				assert.Nil(t, spec)
				return
			}
			assert.Nil(t, err)
			assert.NotNil(t, spec)

			assert.Equal(t, "asyncapi", spec.GetResourceType())
			assert.Equal(t, "kafka://test", spec.GetID())
			assert.Equal(t, "test", spec.GetTitle())
			assert.Equal(t, "1.0.0", spec.GetVersion())

			raw := spec.GetSpecBytes()
			assert.NotEmpty(t, raw)

			endpoints, err := spec.GetEndpoints()
			assert.Nil(t, err)
			assert.NotEmpty(t, endpoints)

			assert.Equal(t, "localhost", endpoints[0].Host)
			assert.Equal(t, int32(9092), endpoints[0].Port)
			assert.Equal(t, "SASL_SSL", endpoints[0].Protocol)
			assert.Equal(t, "kafka", endpoints[0].Routing.Details[protocol].(string))
		})
	}
}
