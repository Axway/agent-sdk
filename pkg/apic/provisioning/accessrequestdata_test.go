package provisioning_test

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic/provisioning"
	"github.com/stretchr/testify/assert"
)

func TestAccessRequestDataBuilder(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Build No Data Credential",
			data: nil,
		},
		{
			name: "Build With Data",
			data: map[string]interface{}{
				"data1": "data1",
				"data2": "data2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := provisioning.NewAccessDataBuilder()

			var accData provisioning.AccessData
			accData = builder.SetData(tt.data)

			assert.NotNil(t, accData)
			assert.Equal(t, tt.data, accData.GetData())
		})
	}
}
