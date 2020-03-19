package transaction

// TypeTransactionSummary - Transaction summary type
const TypeTransactionSummary = "transactionSummary"

// TypeTransactionEvent - Transaction Event type
const TypeTransactionEvent = "transactionEvent"

// SummaryEventProxyIDPrefix - Prefix for proxyID in summary event
const SummaryEventProxyIDPrefix = "remoteApiId_"

// LogEvent - Log event to be sent to Condor
type LogEvent struct {
	Version            string   `json:"version"`
	Stamp              int64    `json:"timestamp"`
	TransactionID      string   `json:"transactionId"`
	Environment        string   `json:"environment,omitempty"`
	APICDeployment     string   `json:"apicDeployment,omitempty"`
	EnvironmentID      string   `json:"environmentId"`
	TenantID           string   `json:"tenantId"`
	TrcbltPartitionID  string   `json:"trcbltPartitionId"`
	Type               string   `json:"type"`
	TransactionEvent   *Event   `json:"transactionEvent,omitempty"`
	TransactionSummary *Summary `json:"transactionSummary,omitempty"`
}

// Summary - Represent the transaction summary event
type Summary struct {
	Status       string `json:"status,omitempty"`
	StatusDetail string `json:"statusDetail,omitempty"`
	Duration     int    `json:"duration,omitempty"`
	Application  string `json:"application,omitempty"`
	Product      string `json:"product,omitempty"`
	Team         string `json:"team,omitempty"`

	Proxy      *Proxy      `json:"proxy,omitempty"`
	EntryPoint *EntryPoint `json:"entryPoint,omitempty"`
}

// Proxy - Represents the proxy definition in summary event
type Proxy struct {
	ID       string `json:"id,omitempty"`
	Revision int    `json:"revision,omitempty"`
	Name     string `json:"name,omitempty"`
}

// EntryPoint - represents the entry point details for API in summary event
type EntryPoint struct {
	Type   string `json:"type,omitempty"`
	Method string `json:"method,omitempty"`
	Path   string `json:"path,omitempty"`
	Host   string `json:"host,omitempty"`
}

// Event - Represents the transaction detail event
type Event struct {
	ID          string    `json:"id,omitempty"`
	ParentID    string    `json:"parentId,omitempty"`
	Source      string    `json:"source,omitempty"`
	Destination string    `json:"destination,omitempty"`
	Duration    int       `json:"duration,omitempty"`
	Direction   string    `json:"direction,omitempty"`
	Status      string    `json:"status,omitempty"`
	Protocol    *Protocol `json:"protocol,omitempty"`
}

// Protocol - Represents the protocol details in transaction detail events
type Protocol struct {
	Type                   string `json:"type,omitempty"`
	URI                    string `json:"uri,omitempty"`
	Args                   string `json:"args,omitempty"`
	Method                 string `json:"method,omitempty"`
	Status                 int    `json:"status,omitempty"`
	StatusText             string `json:"statusText,omitempty"`
	UserAgent              string `json:"userAgent,omitempty"`
	Host                   string `json:"host,omitempty"`
	Version                string `json:"version,omitempty"`
	BytesReceived          int    `json:"bytesReceived,omitempty"`
	BytesSent              int    `json:"bytesSent,omitempty"`
	RemoteName             string `json:"remoteName,omitempty"`
	RemoteAddr             string `json:"remoteAddr,omitempty"`
	RemotePort             int    `json:"remotePort,omitempty"`
	LocalAddr              string `json:"localAddr,omitempty"`
	LocalPort              int    `json:"localPort,omitempty"`
	SslServerName          string `json:"sslServerName,omitempty"`
	SslProtocol            string `json:"sslProtocol,omitempty"`
	SslSubject             string `json:"sslSubject,omitempty"`
	AuthSubjectID          string `json:"authSubjectId,omitempty"`
	RequestHeaders         string `json:"requestHeaders,omitempty"`
	IndexedRequestHeaders  string `json:"indexedRequestHeaders,omitempty"`
	ResponseHeaders        string `json:"responseHeaders,omitempty"`
	IndexedResponseHeaders string `json:"indexedResponseHeaders,omitempty"`
	RequestPayload         string `json:"requestPayload,omitempty"`
	ResponsePayload        string `json:"responsePayload,omitempty"`
	WafStatus              int    `json:"wafStatus,omitempty"`
	Timing                 string `json:"timing,omitempty"`
}
