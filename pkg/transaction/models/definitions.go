package models

// ConsumerDetails  - Represents the consumer details in the transaction summary event
type ConsumerDetails struct {
	Application      *AppDetails       `json:"application,omitempty"` // marketplace application
	PublishedProduct *PublishedProduct `json:"publishedProduct,omitempty"`
	Subscription     *Subscription     `json:"subscription,omitempty"`
}

// Subscription  - Represents the subscription used in transaction summary event
type Subscription struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// PublishedProduct - Represents the product used in the transaction summary event
type PublishedProduct struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// AppDetails - struct for app details to report
type AppDetails struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ConsumerOrgID string `json:"consumerOrgId,omitempty"`
}

// ProviderDetails - Represent the provider details in the transaction summary event
type ProviderDetails struct {
	Application   *AppDetails    `json:"application,omitempty"` // managed application
	Product       *Product       `json:"product,omitempty"`
	ProductPlan   *ProductPlan   `json:"productPlan,omitempty"`
	Quota         *Quota         `json:"quota,omitempty"`
	AssetResource *AssetResource `json:"assetResource,omitempty"`
	API           APIDetails     `json:"api"`
}

// AssetResource  - Represents the asset resource used in transaction summary event
type AssetResource struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Product - Represents the product used in the transaction summary event
type Product struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Quota - Represents the quota used in the transaction summary event
type Quota struct {
	ID string `json:"id,omitempty"`
}

// ProductPlan - Represents the plan used in the transaction summary event
type ProductPlan struct {
	ID string `json:"id,omitempty"`
}

// APIDetails - Represents the api used in the transaction summary event
type APIDetails struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Revision           int    `json:"revision,omitempty"`
	APIServiceInstance string `json:"apiServiceInstance,omitempty"`
}

// DataplaneDetails - Represent the dataplane in the transaction summary event
type DataplaneDetails struct {
	Application *Application `json:"application,omitempty"` // dataplane application
	Product     *Product     `json:"product,omitempty"`     // dataplane product
}

// Application  - Represents the application used in transaction summary event
type Application struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}
