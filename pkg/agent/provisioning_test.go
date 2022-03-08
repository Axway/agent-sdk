package agent

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/apiserver/models/management/v1alpha1"
	"github.com/Axway/agent-sdk/pkg/apic/mock"
	"github.com/stretchr/testify/assert"
)

func TestNewCredentialRequestBuilder(t *testing.T) {
	agent.apicClient = &mock.Client{
		RegisterCredentialRequestDefinitionMock: func(data *v1alpha1.CredentialRequestDefinition, update bool) (*v1alpha1.CredentialRequestDefinition, error) {
			return data, nil
		},
	}

	tests := []struct {
		name         string
		expectedName string
	}{
		{
			name:         "Test APIKey Helper",
			expectedName: "api-key",
		},
		{
			name:         "Test OAuth Helper",
			expectedName: "oauth",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var err error
			var crd *v1alpha1.CredentialRequestDefinition
			switch test.expectedName {
			case "api-key":
				crd, err = NewAPIKeyCredentialRequestBuilder().Register()
			case "oauth":
				crd, err = NewOAuthCredentialRequestBuilder().Register()
			default:
				crd, err = NewCredentialRequestBuilder().Register()
			}
			assert.NotNil(t, crd)
			assert.Nil(t, err)
		})
	}
}
