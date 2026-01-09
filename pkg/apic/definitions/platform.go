package definitions

// SessionEntitlements represents the root structure of the session.json
// which is a list of product entitlements for an org/user session.
type SessionEntitlements struct {
	Success bool                 `json:"success"`
	Result  []EntitlementProduct `json:"result"`
}

type EntitlementProduct struct {
	ID           string             `json:"id"`
	Product      string             `json:"product"`
	Plan         string             `json:"plan"`
	Tier         string             `json:"tier"`
	Governance   string             `json:"governance"`
	StartDate    string             `json:"start_date"`
	EndDate      string             `json:"end_date"`
	Source       string             `json:"source,omitempty"`
	Entitlements []EntitlementEntry `json:"entitlements"`
	Expired      bool               `json:"expired"`
	ProductName  string             `json:"product_name"`
}

type EntitlementEntry struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}

type OrgEntitlementsResponse struct {
	Success bool            `json:"success"`
	Result  OrgEntitlements `json:"result"`
}

// OrgEntitlements contains only the entitlements map for an org.
// Values can be numbers, booleans, or arrays.
type OrgEntitlements struct {
	Entitlements map[string]interface{} `json:"entitlements"`
}
