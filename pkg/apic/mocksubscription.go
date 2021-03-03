package apic

type mockSubscription struct {
	Subscription
	catalogID     string
	serviceClient *ServiceClient
	updateErr     error
}

func (s *mockSubscription) GetID() string                              { return "" }
func (s *mockSubscription) GetName() string                            { return "" }
func (s *mockSubscription) GetApicID() string                          { return "" }
func (s *mockSubscription) GetRemoteAPIID() string                     { return "" }
func (s *mockSubscription) GetRemoteAPIStage() string                  { return "" }
func (s *mockSubscription) GetCatalogItemID() string                   { return s.catalogID }
func (s *mockSubscription) GetCreatedUserID() string                   { return "" }
func (s *mockSubscription) GetState() SubscriptionState                { return SubscriptionApproved }
func (s *mockSubscription) GetServiceClient() *ServiceClient           { return s.serviceClient }
func (s *mockSubscription) GetPropertyValue(propertyKey string) string { return "" }
func (s *mockSubscription) UpdateState(newState SubscriptionState, description string) error {
	return nil
}
func (s *mockSubscription) UpdateEnumProperty(key, value, dataType string) error { return nil }
func (s *mockSubscription) UpdateProperties(appName string) error                { return nil }
func (s *mockSubscription) UpdatePropertyValues(values map[string]interface{}) error {
	return s.updateErr
}
