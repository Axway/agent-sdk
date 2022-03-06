package provisioning

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessRequestBuilder(t *testing.T) {
	tests := []struct {
		name     string
		noSchema bool
		wantErr  bool
	}{
		{
			name:    "Success",
			wantErr: false,
		},
		{
			name:    "Fail",
			wantErr: true,
		},
		{
			name:     "Fail",
			noSchema: true,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registerFuncCalled := false
			// builtDef := struct{}{}
			registerFunc := func(accessRequestDefinition *v1alpha1.AccessRequestDefinition) (*v1alpha1.AccessRequestDefinition, error) {
				// TODO - validate that the accessRequestDefinition is built properly
				// builtDef = accessRequestDefinition.(struct{})
				registerFuncCalled = true
				return nil, nil
			}

			builder := NewAccessRequestBuilder(registerFunc).
				SetName(tt.name)

			if tt.wantErr {
				builder = builder.SetSchema(nil)
			}

			if !tt.noSchema {
				builder.SetSchema(
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
				// assert.NotNil(t, builtDef)
			}
		})
	}
}
