package util

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/definitions"
	"github.com/stretchr/testify/assert"
)

func TestGetAgentDetails(t *testing.T) {
	tests := []struct {
		name     string
		ri       *mockRI
		expected map[string]interface{}
	}{
		{
			name:     "should return nil if no agent details are found",
			ri:       &mockRI{subResources: map[string]interface{}{}},
			expected: nil,
		},
		{
			name: "should return nil if the agent-details key is found, but is not a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]string{},
			}},
			expected: nil,
		},
		{
			name: "should return the agent details sub resource when saved as a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{},
			}},
			expected: map[string]interface{}{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := GetAgentDetails(tc.ri)
			assert.Equal(t, tc.expected, val)
		})
	}
}

func TestGetAgentDetailStrings(t *testing.T) {
	tests := []struct {
		name     string
		ri       *mockRI
		expected map[string]string
	}{
		{
			name:     "should return nil if no agent details are found",
			ri:       &mockRI{subResources: map[string]interface{}{}},
			expected: nil,
		},
		{
			name: "should return nil when the agent-details key is found, but is not a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]string{},
			}},
			expected: nil,
		},
		{
			name: "should return the agent details sub resource when saved as a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{},
			}},
			expected: map[string]string{},
		},
		{
			name: "should map the agent details sub resource as a map[string]string",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					"key1": "val1",
					"key2": []string{"val2a", "val2b"},
				},
			}},
			expected: map[string]string{
				"key1": "val1",
				"key2": "[val2a val2b]",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			val := GetAgentDetailStrings(tc.ri)
			assert.Equal(t, tc.expected, val)
		})
	}
}

func TestGetAgentDetailsValue(t *testing.T) {
	tests := []struct {
		name         string
		ri           *mockRI
		inputKey     string
		err          error
		expectedItem string
		hasError     bool
	}{
		{
			name:         "should return an empty string and nil if x-agent-details is not found",
			ri:           &mockRI{subResources: map[string]interface{}{}},
			inputKey:     "id",
			expectedItem: "",
			hasError:     false,
		},
		{
			name: "should return an empty string and an error if x-agent-details is not a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]string{},
			}},
			inputKey:     "id",
			expectedItem: "",
			hasError:     true,
		},
		{
			name: "should return an empty string and an error if x-agent-details is found, but the key is not found",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{},
			}},
			inputKey:     "id",
			expectedItem: "",
			hasError:     true,
		},
		{
			name: "should return an empty string and an error if x-agent-details is found, and the key is found, but the value is not the correct type",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					"id": map[string]interface{}{},
				},
			}},
			expectedItem: "",
			inputKey:     "id",
			hasError:     true,
		},
		{
			name: "should return the x-agent-details value when the key exists, and the value is a string",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					"id": "123",
				},
			}},
			inputKey:     "id",
			hasError:     false,
			expectedItem: "123",
		},
		{
			name: "should return the x-agent-details value when the key exists, and the value is an int",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{
					"id": 123,
				},
			}},
			inputKey:     "id",
			hasError:     false,
			expectedItem: "123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v, err := GetAgentDetailsValue(tc.ri, tc.inputKey)
			assert.Equal(t, tc.expectedItem, v)

			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestSetAgentDetailsKey(t *testing.T) {
	tests := []struct {
		name     string
		ri       *mockRI
		err      error
		hasError bool
		key      string
		value    interface{}
	}{
		{
			name: "should return an error if x-agent-details is not a map[string]interface{}",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]string{},
			}},
			hasError: true,
			key:      "id",
			value:    "123",
		},
		{
			name:     "should create the x-agent-details sub resource if it does not exist",
			ri:       &mockRI{subResources: map[string]interface{}{}},
			hasError: false,
			key:      "id",
			value:    "123",
		},
		{
			name: "should add the key and value to x-agent-details",
			ri: &mockRI{subResources: map[string]interface{}{
				definitions.XAgentDetails: map[string]interface{}{},
			}},
			hasError: false,
			key:      "id",
			value:    "123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := SetAgentDetailsKey(tc.ri, "id", "123")
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				s, _ := GetAgentDetailsValue(tc.ri, tc.key)
				assert.Equal(t, tc.value, s)
			}
		})
	}
}

func TestSetAgentDetails(t *testing.T) {
	ri := &mockRI{subResources: map[string]interface{}{}}
	SetAgentDetails(ri, map[string]interface{}{})
	assert.Contains(t, ri.subResources, definitions.XAgentDetails)
}

type mockRI struct {
	subResources map[string]interface{}
}

func (m *mockRI) GetSubResource(key string) interface{} {
	if m == nil || m.subResources == nil {
		return nil
	}
	return m.subResources[key]
}

func (m *mockRI) SetSubResource(key string, resource interface{}) {
	if m == nil {
		return
	}

	if m.subResources == nil {
		m.subResources = make(map[string]interface{})
	}
	m.subResources[key] = resource
}
