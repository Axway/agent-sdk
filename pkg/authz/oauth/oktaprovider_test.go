package oauth

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	oktaPKCERequired = "pkce_required"
	oktaSpa          = "okta-spa"
)

func TestOktaPKCERequiredBooleanSerialization(t *testing.T) {
	props := map[string]interface{}{
		oktaPKCERequired: true,
	}
	c, err := NewClientMetadataBuilder().
		SetClientName(oktaSpa).
		SetExtraProperties(props).
		Build()
	assert.Nil(t, err)
	cm := c.(*clientMetadata)

	buf, err := json.Marshal(cm)
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	var out map[string]interface{}
	err = json.Unmarshal(buf, &out)
	assert.Nil(t, err)

	// Should be a boolean, not a string
	val, ok := out[oktaPKCERequired]
	assert.True(t, ok)
	assert.IsType(t, true, val)
	assert.Equal(t, true, val)
}

func TestOktaPKCERequiredBooleanSerializationFalse(t *testing.T) {
	props := map[string]interface{}{
		oktaPKCERequired: false,
	}
	c, err := NewClientMetadataBuilder().
		SetClientName(oktaSpa).
		SetExtraProperties(props).
		Build()
	assert.Nil(t, err)
	cm := c.(*clientMetadata)

	buf, err := json.Marshal(cm)
	assert.Nil(t, err)
	assert.NotNil(t, buf)

	var out map[string]interface{}
	err = json.Unmarshal(buf, &out)
	assert.Nil(t, err)

	// Should be a boolean, not a string
	val, ok := out[oktaPKCERequired]
	assert.True(t, ok)
	assert.IsType(t, false, val)
	assert.Equal(t, false, val)
}
