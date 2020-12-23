package apic

// MockTokenGetter - this is for use in unit tests to bypass the actual tokengetter`
type mockTokenGetter struct {
}

// MockTokenGetter - global var for use in unit tests to return a fake token
var MockTokenGetter = &mockTokenGetter{}

// GetToken -
func (m *mockTokenGetter) GetToken() (string, error) {
	return "testToken", nil
}

// Close -
func (m *mockTokenGetter) Close() error { return nil }
