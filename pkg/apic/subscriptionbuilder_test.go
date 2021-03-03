package apic

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSubscriptionBuilder(t *testing.T) {
	mockSubscription := &mockSubscription{}
	builder := NewSubscriptionBuilder(mockSubscription)
	err := builder.Process()
	assert.Nil(t, err)

	// test all the default values
	assert.Nil(t, builder.(*subscriptionBuilder).err)
	assert.Len(t, builder.(*subscriptionBuilder).propertyValues, 0)
	assert.Equal(t, mockSubscription, builder.(*subscriptionBuilder).subscription)
}

func TestSubscriptionBuilderFuncs(t *testing.T) {
	subscription := &mockSubscription{}
	builder := NewSubscriptionBuilder(subscription).
		SetStringPropertyValue("key1", "value1").
		SetStringPropertyValue("key1", "value2")

	err := builder.Process()
	assert.NotNil(t, err)

	subscription = &mockSubscription{
		updateErr: fmt.Errorf("error"),
	}
	builder = NewSubscriptionBuilder(subscription).
		SetStringPropertyValue("key1", "value1").
		SetStringPropertyValue("key2", "value2")

	err = builder.Process()
	assert.NotNil(t, err)

	testServiceClient, _ := GetTestServiceClient()
	subscription = &mockSubscription{
		catalogID:     "1234",
		serviceClient: testServiceClient,
	}
	builder = NewSubscriptionBuilder(subscription).
		UpdateEnumProperty("appName", "value1", "string").
		SetStringPropertyValue("appName", "value1").
		SetStringPropertyValue("appID", "value2")

	err = builder.Process()
	// check builder properties
	assert.Len(t, builder.(*subscriptionBuilder).propertyValues, 2)
	assert.Equal(t, "value1", builder.(*subscriptionBuilder).propertyValues["appName"])
	assert.Equal(t, "value2", builder.(*subscriptionBuilder).propertyValues["appID"])
	assert.Nil(t, err)

	subscription = &mockSubscription{
		catalogID:     "1234",
		serviceClient: testServiceClient,
	}

	err = NewSubscriptionBuilder(subscription).
		UpdateEnumProperty("appName", "value1", "string").
		SetStringPropertyValue("appName", "value1").
		SetStringPropertyValue("appID", "value2").
		Process()
	assert.Nil(t, err)
}
