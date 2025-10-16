package oauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOktaPKCERequiredBooleanSerialization(t *testing.T) {
	props := map[string]string{
		OktaPKCERequired: "true",
	}
	c, err := NewClientMetadataBuilder().
		SetClientName("okta-spa").
		SetExtraProperties(props).
		Build()
	assert.Nil(t, err)
	cm := c.(*clientMetadata)

	// Simulate OKTA provider logic
	cm.extraProperties[OktaPKCERequired+SuffixBool] = cm.extraProperties[OktaPKCERequired]
	delete(cm.extraProperties, OktaPKCERequired)

	buf, err := json.Marshal(cm)
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	var out map[string]interface{}
	err = json.Unmarshal(buf, &out)
	assert.Nil(t, err)

	// Should be a boolean, not a string
	val, ok := out[OktaPKCERequired]
	assert.True(t, ok)
	assert.IsType(t, true, val)
	assert.Equal(t, true, val)
}

func TestOktaPKCERequiredBooleanSerializationFalse(t *testing.T) {
	props := map[string]string{
		OktaPKCERequired: "false",
	}
	c, err := NewClientMetadataBuilder().
		SetClientName("okta-spa").
		SetExtraProperties(props).
		Build()
	assert.Nil(t, err)
	cm := c.(*clientMetadata)

	// Simulate OKTA provider logic
	cm.extraProperties[OktaPKCERequired+SuffixBool] = cm.extraProperties[OktaPKCERequired]
	delete(cm.extraProperties, OktaPKCERequired)

	buf, err := json.Marshal(cm)
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	var out map[string]interface{}
	err = json.Unmarshal(buf, &out)
	assert.Nil(t, err)

	// Should be a boolean, not a string
	val, ok := out[OktaPKCERequired]
	assert.True(t, ok)
	assert.IsType(t, false, val)
	assert.Equal(t, false, val)
}
