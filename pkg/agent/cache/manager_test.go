package cache

import (
	"testing"

	"github.com/Axway/agent-sdk/pkg/apic"
	v1 "github.com/Axway/agent-sdk/pkg/apic/apiserver/models/api/v1"
	"github.com/Axway/agent-sdk/pkg/config"
	"github.com/stretchr/testify/assert"
)

func createAPIService(apiID, apiName, primaryKey string) *v1.ResourceInstance {
	attributes := map[string]string{
		apic.AttrExternalAPIID:   apiID,
		apic.AttrExternalAPIName: apiName,
	}
	if primaryKey != "" {
		attributes[apic.AttrExternalAPIPrimaryKey] = primaryKey
	}

	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Attributes: attributes,
		},
	}
}

func createAPIServiceInstance(id string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Metadata: v1.Metadata{
				ID: id,
			},
		},
	}
}

func createCategory(name, title string) *v1.ResourceInstance {
	return &v1.ResourceInstance{
		ResourceMeta: v1.ResourceMeta{
			Name:  name,
			Title: title,
		},
	}
}

// add api service with externalAPIID, externalAPIName
// add api service with externalAPIPrimaryKey, externalAPIID, externalAPIName
// add existing api service with externalAPIID, externalAPIName
// get api service with APIID added by externalAPIID
// get api service with APIID added by externalAPIPrimaryKey
// get api service with Name added by externalAPIID
// get api service with Name added by externalAPIPrimaryKey
// get api service with invalid primary key
// get api service with primary key added by externalAPIPrimaryKey
// delete api service with APIID added by externalAPIID
// delete api service with APIID added by externalAPIPrimaryKey
func TestAPIServiceCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{})
	assert.NotNil(t, m)

	apiCache := m.GetAPIServiceCache()
	assert.NotNil(t, apiCache)
	assert.Equal(t, apiCache.GetKeys(), m.GetAPIServiceKeys())

	api1 := createAPIService("id1", "api1", "")
	api2 := createAPIService("id2", "api2", "api2key")

	externalAPIID := m.AddAPIService(api1)
	assert.Equal(t, "id1", externalAPIID)
	externalAPIID = m.AddAPIService(api2)
	assert.Equal(t, "id2", externalAPIID)
	externalAPIID = m.AddAPIService(api2)
	assert.Equal(t, "id2", externalAPIID)

	cachedAPI := m.GetAPIServiceWithAPIID("id1")
	assert.Equal(t, api1, cachedAPI)

	cachedAPI = m.GetAPIServiceWithAPIID("id2")
	assert.Equal(t, api2, cachedAPI)

	cachedAPI = m.GetAPIServiceWithName("api1")
	assert.Equal(t, api1, cachedAPI)

	cachedAPI = m.GetAPIServiceWithName("api2")
	assert.Equal(t, api2, cachedAPI)

	cachedAPI = m.GetAPIServiceWithPrimaryKey("api1key")
	assert.Nil(t, cachedAPI)

	cachedAPI = m.GetAPIServiceWithPrimaryKey("api2key")
	assert.Equal(t, api2, cachedAPI)

	err := m.DeleteAPIService("api1")
	assert.Nil(t, err)
	cachedAPI = m.GetAPIServiceWithAPIID("api1")
	assert.Nil(t, cachedAPI)

	err = m.DeleteAPIService("api2")
	assert.Nil(t, err)
	cachedAPI = m.GetAPIServiceWithAPIID("api2")
	assert.Nil(t, cachedAPI)

	err = m.DeleteAPIService("api2")
	assert.NotNil(t, err)
}

// add api service instance
// get api service instance by ID
// get api service instance with invalid ID
// delete api service instance by ID
// delete all api service instance
func TestAPIServiceInstanceCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{})
	assert.NotNil(t, m)
	assert.Equal(t, []string{}, m.GetAPIServiceKeys())

	instance1 := createAPIServiceInstance("id1")
	instance2 := createAPIServiceInstance("id2")

	m.AddAPIServiceInstance(instance1)
	m.AddAPIServiceInstance(instance2)
	assert.ElementsMatch(t, []string{"id1", "id2"}, m.GetAPIServiceInstanceKeys())

	cachedInstance, err := m.GetAPIServiceInstanceByID("id1")
	assert.Nil(t, err)
	assert.Equal(t, instance1, cachedInstance)

	err = m.DeleteAPIServiceInstance("id1")
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"id2"}, m.GetAPIServiceInstanceKeys())

	cachedInstance, err = m.GetAPIServiceInstanceByID("id1")
	assert.NotNil(t, err)
	assert.Nil(t, cachedInstance)

	m.DeleteAllAPIServiceInstance()
	assert.ElementsMatch(t, []string{}, m.GetAPIServiceInstanceKeys())
}

// add category
// get category with name
// get category with title
// get category with invalid name
// delete category
func TestCategoryCache(t *testing.T) {
	m := NewAgentCacheManager(&config.CentralConfiguration{})
	assert.NotNil(t, m)

	categoryCache := m.GetCategoryCache()
	assert.NotNil(t, categoryCache)

	assert.Equal(t, []string{}, m.GetCategoryKeys())

	category1 := createCategory("c1", "category 1")
	category2 := createCategory("c2", "category 2")

	m.AddCategory(category1)
	assert.ElementsMatch(t, []string{"c1"}, m.GetCategoryKeys())
	m.AddCategory(category2)
	assert.ElementsMatch(t, []string{"c1", "c2"}, m.GetCategoryKeys())

	cachedCategory := m.GetCategory("c1")
	assert.Equal(t, category1, cachedCategory)

	cachedCategory = m.GetCategoryWithTitle("category 2")
	assert.Equal(t, category2, cachedCategory)

	err := m.DeleteCategory("c1")
	assert.Nil(t, err)
	assert.ElementsMatch(t, []string{"c2"}, m.GetCategoryKeys())

	cachedCategory = m.GetCategory("c1")
	assert.Nil(t, cachedCategory)

	err = m.DeleteCategory("c1")
	assert.NotNil(t, err)
}
