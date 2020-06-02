package apic

// APIServerInfoProperty -
type APIServerInfoProperty struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// APIServerInfo -
type APIServerInfo struct {
	ConsumerInstance APIServerInfoProperty `json:"consumerInstance,omitempty"`
	Environment      APIServerInfoProperty `json:"environment,omitempty"`
}

// APIServerScope -
type APIServerScope struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
}

// APIServerReference -
type APIServerReference struct {
	ID   string `json:"id,omitempty"`
	Kind string `json:"kind,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// APIServerMetadata -
type APIServerMetadata struct {
	ID         string               `json:"id,omitempty"`
	Scope      *APIServerScope      `json:"scope,omitempty"`
	References []APIServerReference `json:"references,omitempty"`
}

// APIServer -
type APIServer struct {
	Name       string                 `json:"name"`
	Title      string                 `json:"title"`
	Tags       []string               `json:"tags"`
	Attributes map[string]interface{} `json:"attributes"`
	Spec       interface{}            `json:"spec"`
	Metadata   *APIServerMetadata     `json:"metadata,omitempty"`
}

// APIServiceSpec -
type APIServiceSpec struct {
	Description string          `json:"description"`
	Icon        *APIServiceIcon `json:"icon,omitempty"`
}

// APIServiceRevisionSpec -
type APIServiceRevisionSpec struct {
	APIService string             `json:"apiService"`
	Definition RevisionDefinition `json:"definition"`
}

// RevisionDefinition -
type RevisionDefinition struct {
	Type  string `json:"type,omitempty"`
	Value []byte `json:"value,omitempty"`
}

// APIServiceIcon -
type APIServiceIcon struct {
	ContentType string `json:"contentType"`
	Data        string `json:"data"`
}

// APIServerInstanceSpec -
type APIServerInstanceSpec struct {
	APIServiceRevision string     `json:"apiServiceRevision,omitempty"`
	InstanceEndPoint   []EndPoint `json:"endpoint,omitempty"`
}

// EndPoint -
type EndPoint struct {
	Host     string   `json:"host,omitempty"`
	Port     int      `json:"port,omitempty"`
	Protocol string   `json:"protocol,omitempty"`
	Routing  BasePath `json:"routing,omitempty"`
}

// BasePath -
type BasePath struct {
	Path string `json:"basePath,omitempty"`
}

//EnvironmentSpec - structure of environment returned when not using API Server
type EnvironmentSpec struct {
	ID       string      `json:"id,omitempty"`
	Name     string      `json:"name,omitempty"`
	Metadata interface{} `json:"metadata,omitempty"`
}

// ConsumerInstanceSpec -
type ConsumerInstanceSpec struct {
	Name               string          `json:"name,omitempty"`
	APIServiceInstance string          `json:"apiServiceInstance"`
	OwningTeam         string          `json:"owningTeam,omitempty"`
	Description        string          `json:"description,omitempty"`
	Visibility         string          `json:"visibility,omitempty"` // default: RESTRICTED
	Version            string          `json:"version,omitempty"`
	State              string          `json:"state,omitempty"` // default: UNPUBLISHED
	Status             string          `json:"status,omitempty"`
	Tags               []string        `json:"tags,omitempty"`
	Icon               *APIServiceIcon `json:"icon,omitempty"`
	Documentation      string          `json:"documentation,omitempty"`

	// UnstructuredDataProperties *APIServiceSubscription `json:"subscription"`
	// AdditionalDataProperties *APIServiceSubscription `json:"subscription"`
	Subscription *APIServiceSubscription `json:"subscription"`
}

//APIServiceSubscription -
type APIServiceSubscription struct {
	Enabled                bool   `json:"enabled,omitempty"`       // default: false
	AutoSubscribe          bool   `json:"autoSubscribe,omitempty"` // default: true
	SubscriptionDefinition string `json:"subscriptionDefinition"`
}
