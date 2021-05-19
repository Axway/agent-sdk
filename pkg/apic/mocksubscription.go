package apic

import "fmt"

// MockSubscription - use for ease of testing agents
type MockSubscription struct {
	Subscription
	ID                           string
	Description                  string
	Name                         string
	ApicID                       string
	RemoteAPIID                  string
	RemoteAPIStage               string
	CatalogID                    string
	UserID                       string
	State                        SubscriptionState
	PropertyVals                 map[string]string
	ReceivedValues               map[string]interface{}
	RemoteAPIAttributes          map[string]string
	ReceivedAppName              string
	ReceivedUpdatedEnum          string
	UpdateStateErr               error
	UpdateEnumErr                error
	UpdatePropertiesErr          error
	UpdatePropertyValErr         error
	UpdateStateWithPropertiesErr error
}

//GetID - mocked for testing
func (s *MockSubscription) GetID() string { return s.ID }

//GetName - mocked for testing
func (s *MockSubscription) GetName() string { return s.Name }

//GetApicID - mocked for testing
func (s *MockSubscription) GetApicID() string { return s.ApicID }

//GetRemoteAPIID - mocked for testing
func (s *MockSubscription) GetRemoteAPIID() string { return s.RemoteAPIID }

// GetRemoteAPIAttributes - mocked for testing
func (s *MockSubscription) GetRemoteAPIAttributes() map[string]string { return s.RemoteAPIAttributes }

//GetRemoteAPIStage - mocked for testing
func (s *MockSubscription) GetRemoteAPIStage() string { return s.RemoteAPIStage }

//GetCatalogItemID - mocked for testing
func (s *MockSubscription) GetCatalogItemID() string { return s.CatalogID }

//GetCreatedUserID - mocked for testing
func (s *MockSubscription) GetCreatedUserID() string { return s.UserID }

//GetState - mocked for testing
func (s *MockSubscription) GetState() SubscriptionState { return s.State }

//GetPropertyValue - mocked for testing
func (s *MockSubscription) GetPropertyValue(propertyKey string) string {
	return s.PropertyVals[propertyKey]
}

//UpdateState - mocked for testing
func (s *MockSubscription) UpdateState(newState SubscriptionState, description string) error {
	if s.UpdateStateErr == nil {
		s.State = newState
		s.Description = description
	}
	return s.UpdateStateErr
}

//UpdateEnumProperty - mocked for testing
func (s *MockSubscription) UpdateEnumProperty(key, value, dataType string) error {
	if s.UpdateEnumErr == nil {
		s.ReceivedUpdatedEnum = fmt.Sprintf("%v---%v---%v", key, value, dataType)
	}
	return s.UpdateEnumErr
}

//UpdateProperties - mocked for testing
func (s *MockSubscription) UpdateProperties(appName string) error {
	if s.UpdatePropertiesErr == nil {
		s.ReceivedAppName = appName
	}
	return s.UpdatePropertiesErr
}

//UpdatePropertyValues - mocked for testing
func (s *MockSubscription) UpdatePropertyValues(values map[string]interface{}) error {
	if s.UpdatePropertyValErr == nil {
		s.ReceivedValues = values
	}
	return s.UpdatePropertyValErr
}

// UpdateStateWithProperties - mocked for testing
func (s *MockSubscription) UpdateStateWithProperties(newState SubscriptionState, _ string, props map[string]interface{}) error {
	if s.UpdateStateWithPropertiesErr == nil {
		s.State = newState
		s.ReceivedValues = props
		return nil
	}
	return s.UpdateStateWithPropertiesErr
}
