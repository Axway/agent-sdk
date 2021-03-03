package apic

import (
	"fmt"
)

//SubscriptionBuilder -
type SubscriptionBuilder interface {
	UpdateEnumProperty(key, newValue, dataType string) SubscriptionBuilder
	SetStringPropertyValue(key, value string) SubscriptionBuilder
	Process() error
}

type subscriptionBuilder struct {
	err            error
	subscription   Subscription
	propertyValues map[string]interface{}
}

// NewSubscriptionBuilder - Creates a new subscription builder to update a subscriptions property and status
func NewSubscriptionBuilder(subscription Subscription) SubscriptionBuilder {
	return &subscriptionBuilder{
		subscription:   subscription,
		propertyValues: make(map[string]interface{}),
	}
}

// Process - use the subscription property values and update the subscription on API Central
func (s *subscriptionBuilder) Process() error {
	if s.err != nil {
		return s.err
	}
	return s.subscription.UpdatePropertyValues(s.propertyValues)
}

// UpdateEnumProperty - updates the catalog item with the new value that should be in the enum
func (s *subscriptionBuilder) UpdateEnumProperty(key, newValue, dataType string) SubscriptionBuilder {
	err := s.subscription.UpdateEnumProperty(key, newValue, dataType)
	if err != nil {
		s.err = err
	}

	return s
}

// SetStringPropertyValue - save the key/value pair to the map for the value in this subscription
func (s *subscriptionBuilder) SetStringPropertyValue(key, value string) SubscriptionBuilder {
	if _, ok := s.propertyValues[key]; ok {
		s.err = fmt.Errorf("Key %v already had a value, not updated", key)
	} else {
		s.propertyValues[key] = value
	}
	return s
}
