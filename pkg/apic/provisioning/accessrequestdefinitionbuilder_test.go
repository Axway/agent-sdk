package provisioning

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessRequestBuilder(t *testing.T) {
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
			registerFunc := func(accessRequestDefinition *v1alpha1.AccessRequestDefinition) (*v1alpha1.AccessRequestDefinition, error) {
				assert.NotNil(t, accessRequestDefinition)
				if !tt.noSchema {
					assert.Len(t, accessRequestDefinition.Spec.Schema["properties"], 1)
					assert.NotNil(t, accessRequestDefinition.Spec.Schema["properties"].(map[string]interface{})["prop"])
				} else {
					assert.Len(t, accessRequestDefinition.Spec.Schema["properties"], 0)
				}
				registerFuncCalled = true
				return nil, nil
			}

			builder := NewAccessRequestBuilder(registerFunc).
				SetName(tt.name)

			if tt.wantErr {
				builder = builder.SetRequestSchema(nil)
				builder = builder.SetProvisionSchema(nil)
			}

			if !tt.noSchema {
				b := builder.
					SetRequestSchema(
						NewSchemaBuilder().
							SetName("schema").
							AddProperty(
								NewSchemaPropertyBuilder().
									SetName("prop").
									IsString()))
				if tt.copySchema {
					b.SetProvisionSchemaToRequestSchema()
				}
				b.SetProvisionSchema(
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
