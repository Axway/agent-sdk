package apic

// MockSubscriptionManager - used for unit tests to bypass the normal SubscriptionManager
type MockSubscriptionManager struct {
	SubscriptionManager
	RegisterProcessorCalled int
	RegisterValidatorCalled int
	StartCalled             int
	StopCalled              int
}

// NewMockSubscriptionManager -
func NewMockSubscriptionManager() *MockSubscriptionManager {
	return &MockSubscriptionManager{}
}

// RegisterProcessor -
func (m *MockSubscriptionManager) RegisterProcessor(state SubscriptionState, processor SubscriptionProcessor) {
	m.RegisterProcessorCalled++
}

// RegisterValidator -
func (m *MockSubscriptionManager) RegisterValidator(validator SubscriptionValidator) {
	m.RegisterValidatorCalled++
}

// Start -
func (m *MockSubscriptionManager) Start() {
	m.StartCalled++
}

// Stop -
func (m *MockSubscriptionManager) Stop() {
	m.StopCalled++
}
