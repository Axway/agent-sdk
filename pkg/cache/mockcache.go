package cache

type MockCache struct {
}

func (m MockCache) Get(_ string) (interface{}, error) {
	return nil, nil
}

func (m MockCache) GetItem(_ string) (*Item, error) {
	return nil, nil
}

func (m MockCache) GetBySecondaryKey(_ string) (interface{}, error) {
	return nil, nil
}

func (m MockCache) GetItemBySecondaryKey(_ string) (*Item, error) {
	return nil, nil
}

func (m MockCache) GetForeignKeys() []string {
	return nil
}

func (m MockCache) GetItemsByForeignKey(_ string) ([]*Item, error) {
	return nil, nil
}

func (m MockCache) GetKeys() []string {
	return nil
}

func (m MockCache) HasItemChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

func (m MockCache) HasItemBySecondaryKeyChanged(_ string, _ interface{}) (bool, error) {
	return false, nil
}

func (m MockCache) Set(_ string, _ interface{}) error {
	return nil
}

func (m MockCache) SetWithSecondaryKey(_ string, _ string, _ interface{}) error {
	return nil
}

func (m MockCache) SetWithForeignKey(_ string, _ string, _ interface{}) error {
	return nil
}

func (m MockCache) SetSecondaryKey(_ string, _ string) error {
	return nil
}

func (m MockCache) SetForeignKey(_ string, _ string) error {
	return nil
}

func (m MockCache) Delete(_ string) error {
	return nil
}

func (m MockCache) DeleteBySecondaryKey(_ string) error {
	return nil
}

func (m MockCache) DeleteSecondaryKey(_ string) error {
	return nil
}

func (m MockCache) DeleteForeignKey(_ string) error {
	return nil
}

func (m MockCache) DeleteItemsByForeignKey(_ string) error {
	return nil
}

func (m MockCache) Flush() {
}

func (m MockCache) Save(_ string) error {
	return nil
}

func (m MockCache) Load(_ string) error {
	return nil
}
