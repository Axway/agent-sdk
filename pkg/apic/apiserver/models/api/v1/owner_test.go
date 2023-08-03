package v1

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOwner_MarshalJSON(t *testing.T) {
	o := &Owner{}
	o.SetID("123")

	b, err := o.MarshalJSON()
	assert.Nil(t, err)

	o2 := &Owner{}
	err = json.Unmarshal(b, o2)
	assert.Nil(t, err)
	assert.Equal(t, o.Type, o2.Type)
	assert.Equal(t, o.ID, o2.ID)

	o = &Owner{}
	o.SetType(TeamOwner)
	o.SetID("123")

	b, err = o.MarshalJSON()
	assert.Nil(t, err)

	o2 = &Owner{}
	err = json.Unmarshal(b, o2)
	assert.Nil(t, err)
	assert.Equal(t, o.Type, o2.Type)
	assert.Equal(t, o.ID, o2.ID)

	invalid := []byte(`{"type":"fake","id":"123"}`)
	err = json.Unmarshal(invalid, o2)
	assert.NotNilf(t, err, "should fail when given an invalid type")

	validNoOwnerType := []byte(`{"id":"123"}`)
	err = json.Unmarshal(validNoOwnerType, o2)
	assert.Nil(t, err)
	assert.Equal(t, o.Type, TeamOwner)
	assert.Equal(t, o.ID, o2.ID)
}
