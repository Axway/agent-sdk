package apic

type MockSubscriptionManager struct {
	SubscriptionManager
	RegisterProcessorCalled   int
	RegisterValidatorCalled   int
	StartCalled               int
	StopCalled                int
	AddBlacklistItemCalled    int
	RemoveBlacklistItemCalled int
}

// NewMockSubscriptionManager -
func NewMockSubscriptionManager() *MockSubscriptionManager {
	return &MockSubscriptionManager{}
}

func (m *MockSubscriptionManager) RegisterProcessor(state SubscriptionState, processor SubscriptionProcessor) {
	m.RegisterProcessorCalled++
}

func (m *MockSubscriptionManager) RegisterValidator(validator SubscriptionValidator) {
	m.RegisterValidatorCalled++
}

func (m *MockSubscriptionManager) Start() {
	m.StartCalled++
}

func (m *MockSubscriptionManager) Stop() {
	m.StopCalled++
}

func (m *MockSubscriptionManager) AddBlacklistItem(id string) {
	m.AddBlacklistItemCalled++
}

func (m *MockSubscriptionManager) RemoveBlacklistItem(id string) {
	m.RemoveBlacklistItemCalled++
}
