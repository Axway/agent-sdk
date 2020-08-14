package apic

// MockTokenGetter - this is for use in unit tests to bypass the actual tokengetter`
type MockTokenGetter struct {
}

// GetToken -
func (m *MockTokenGetter) GetToken() (string, error) {
	return "testToken", nil
}
