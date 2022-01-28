package cache

// MockCache a mock cache
type MockCache struct {
}

// Get -
func (m MockCache) Get(_ string) (interface{}, error) {
	return nil, nil
}

// GetItem -
func (m MockCache) GetItem(_ string) (*Item, error) {
	return nil, nil
}

// GetBySecondaryKey -
func (m MockCache) GetBySecondaryKey(_ string) (interface{}, error) {
	return nil, nil
}

// GetItemBySecondaryKey -
func (m MockCache) GetItemBySecondaryKey(_ string) (*Item, error) {
	return nil, nil
}

// GetForeignKeys -
func (m MockCache) GetForeignKeys() []string {
	return nil
}

// GetItemsByForeignKey -
func (m MockCache) GetItemsByForeignKey(_ string) ([]*Item, error) {
	return nil, nil
}

// GetKeys -
func (m MockCache) GetKeys() []string {
	return nil
}

// HasItemChanged -
func (m MockCache) HasItemChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

// HasItemBySecondaryKeyChanged -
func (m MockCache) HasItemBySecondaryKeyChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

// Set -
func (m MockCache) Set(_ string, _ interface{}) error {
	return nil
}

// SetWithSecondaryKey -
func (m MockCache) SetWithSecondaryKey(_ string, _ string, _ interface{}) error {
	return nil
}

// SetWithForeignKey -
func (m MockCache) SetWithForeignKey(_ string, _ string, _ interface{}) error {
	return nil
}

// SetSecondaryKey -
func (m MockCache) SetSecondaryKey(_ string, _ string) error {
	return nil
}

// SetForeignKey -
func (m MockCache) SetForeignKey(_ string, _ string) error {
	return nil
}

// Delete -
func (m MockCache) Delete(_ string) error {
	return nil
}

// DeleteBySecondaryKey -
func (m MockCache) DeleteBySecondaryKey(_ string) error {
	return nil
}

// DeleteSecondaryKey -
func (m MockCache) DeleteSecondaryKey(_ string) error {
	return nil
}

// DeleteForeignKey -
func (m MockCache) DeleteForeignKey(_ string) error {
	return nil
}

// DeleteItemsByForeignKey -
func (m MockCache) DeleteItemsByForeignKey(_ string) error {
	return nil
}

// Flush -
func (m MockCache) Flush() {
}

// Save -
func (m MockCache) Save(_ string) error {
	return nil
}

// Load -
func (m MockCache) Load(_ string) error {
	return nil
}
