package models

// ConsumerDetails  - Represents the consumer details in the transaction summary event
type ConsumerDetails struct {
	Application      *AppDetails   `json:"application,omitempty"` // marketplace application
	PublishedProduct *Product      `json:"publishedProduct,omitempty"`
	Subscription     *Subscription `json:"subscription,omitempty"`
}

// Subscription  - Represents the subscription used in transaction summary consumer details
type Subscription struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// AppDetails - struct for app details to report
type AppDetails struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ConsumerOrgID string `json:"consumerOrgId,omitempty"`
}

// AssetResource  - Represents the asset resource used in transaction summary provider details event
type AssetResource struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Product - Represents the product used in the transaction summary provider details event
type Product struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

// Quota - Represents the quota used in the transaction summary provider details event
type Quota struct {
	ID string `json:"id,omitempty"`
}

// ProductPlan - Represents the plan used in the transaction summary provider details event
type ProductPlan struct {
	ID string `json:"id,omitempty"`
}

// APIDetails - Represents the api used in the transaction summary provider details event
type APIDetails struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Revision           int    `json:"revision,omitempty"`
	TeamID             string `json:"teamID,omitempty"`
	APIServiceInstance string `json:"apiServiceInstance,omitempty"`
}
