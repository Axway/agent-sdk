package provisioning

import (
	"testing"

	management "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNewApplicationProfileBuilder(t *testing.T) {
	tests := []struct {
		name       string
		noSchema   bool
		copySchema bool
		wantErr    bool
	}{
		{
			name: "Success",
		},
		{
			name:    "Fail",
			wantErr: true,
		},
		{
			name:     "Empty",
			noSchema: true,
		},
		{
			name:       "Copied",
			copySchema: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerFuncCalled := false
			registerFunc := func(applicationProfileDefinition *management.ApplicationProfileDefinition) (*management.ApplicationProfileDefinition, error) {
				assert.NotNil(t, applicationProfileDefinition)
				if !tt.noSchema {
					assert.Len(t, applicationProfileDefinition.Spec.Schema["properties"], 1)
					assert.NotNil(t, applicationProfileDefinition.Spec.Schema["properties"].(map[string]interface{})["prop"])
				} else {
					assert.Len(t, applicationProfileDefinition.Spec.Schema["properties"], 0)
				}
				registerFuncCalled = true
				return nil, nil
			}

			builder := NewApplicationProfileBuilder(registerFunc).
				SetName(tt.name)

			if tt.wantErr {
				builder = builder.SetRequestSchema(nil)
			}

			if !tt.noSchema {
				builder = builder.
					SetRequestSchema(
						NewSchemaBuilder().
							SetName("schema").
							AddProperty(
								NewSchemaPropertyBuilder().
									SetName("prop").
									IsString()))
			}

			_, err := builder.Register()

			if tt.wantErr {
				assert.NotNil(t, err)
				assert.False(t, registerFuncCalled)
			} else {
				assert.Nil(t, err)
				assert.True(t, registerFuncCalled)
			}
		})
	}
}
