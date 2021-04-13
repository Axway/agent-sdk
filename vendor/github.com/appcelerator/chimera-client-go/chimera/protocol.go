package chimera

// Chimera requests protocols, default to HTTPS.
type Scheme int

const (
	HTTPS Scheme = iota
	HTTP
)

var protocols = [...]string{
	"https",
	"http",
}

func (s Scheme) String() string { return protocols[s] }
