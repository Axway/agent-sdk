package watchmanager

type TokenGetter func() (string, error)

type Config struct {
	ScopeKind  string
	Scope      string
	Group      string
	Kind       string
	Name       string
	EventTypes []string
}
