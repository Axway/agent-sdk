package provisioning

// AccessData - holds the details about the access data to send to platform
type AccessData interface {
	GetData() map[string]interface{}
}

type accessData struct {
	AccessData
	data map[string]interface{}
}

func (c accessData) GetData() map[string]interface{} {
	return c.data
}

// AccessDataBuilder - builder to create new access data to send to Central
type AccessDataBuilder interface {
	SetData(data map[string]interface{}) AccessData
}

type accessDataBuilder struct {
	access *accessData
}

// NewAccessDataBuilder - create a access data builder
func NewAccessDataBuilder() AccessDataBuilder {
	return &accessDataBuilder{
		access: &accessData{},
	}
}

// SetCredential - set the access data
func (a *accessDataBuilder) SetData(data map[string]interface{}) AccessData {
	a.access.data = data
	return a.access
}
