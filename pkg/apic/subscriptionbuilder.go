package apic

import (
	"fmt"

	agenterrors "github.com/Axway/agent-sdk/pkg/util/errors"
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
	catalogItemID := s.subscription.GetCatalogItemID()

	// First need to get the subscriptionDefProperties for the catalog item
	ss, err := s.subscription.GetServiceClient().GetSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey)
	if ss == nil || err != nil {
		s.err = agenterrors.Wrap(ErrGetSubscriptionDefProperties, err.Error())
		return s
	}

	// update the appName in the enum
	prop := ss.GetProperty(appNameKey)
	newOptions := append(prop.Enum, newValue)
	ss.AddProperty(key, dataType, "", "", true, newOptions)

	// update the the subscriptionDefProperties for the catalog item. This MUST be done before updating the subscription
	err = s.subscription.GetServiceClient().UpdateSubscriptionDefinitionPropertiesForCatalogItem(catalogItemID, profileKey, ss)
	if err != nil {
		s.err = agenterrors.Wrap(ErrUpdateSubscriptionDefProperties, err.Error())
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
