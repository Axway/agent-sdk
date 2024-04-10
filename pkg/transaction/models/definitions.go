package models

import "github.com/sirupsen/logrus"

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

func (a Subscription) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["subscriptionID"] = a.ID
	}
	return fields
}

// AppDetails - struct for app details to report
type AppDetails struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ConsumerOrgID string `json:"consumerOrgId,omitempty"`
}

func (a AppDetails) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["appID"] = a.ID
	}
	return fields
}

// AssetResource  - Represents the asset resource used in transaction summary provider details event
type AssetResource struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

func (a AssetResource) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["assetID"] = a.ID
	}
	return fields
}

// Product - Represents the product used in the transaction summary provider details event
type Product struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	VersionName string `json:"versionName,omitempty"`
	VersionID   string `json:"versionId,omitempty"`
}

func (a Product) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["productID"] = a.ID
		fields["productVersionID"] = a.VersionID
	}
	return fields
}

// Quota - Represents the quota used in the transaction summary provider details event
type Quota struct {
	ID string `json:"id,omitempty"`
}

func (a Quota) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["quotaID"] = a.ID
	}
	return fields
}

// ProductPlan - Represents the plan used in the transaction summary provider details event
type ProductPlan struct {
	ID string `json:"id,omitempty"`
}

func (a ProductPlan) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["planID"] = a.ID
	}
	return fields
}

// APIDetails - Represents the api used in the transaction summary provider details event
type APIDetails struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Revision           int    `json:"revision,omitempty"`
	TeamID             string `json:"teamId,omitempty"`
	APIServiceInstance string `json:"apiServiceInstance,omitempty"`
	Stage              string `json:"-"`
	Version            string `json:"-"`
}

func (a APIDetails) GetLogFields(fields logrus.Fields) logrus.Fields {
	if a.ID != "unknown" {
		fields["apiID"] = a.ID
	}
	return fields
}
