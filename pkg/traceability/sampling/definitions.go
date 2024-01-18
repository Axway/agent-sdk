package sampling

// SampleKey - the key used in the metadata when a transaction qualifies for sampling and should be sent to Observer
// defaultSamplingRate - the default sampling rate in percentage
const (
	SampleKey           = "sample"
	countMax            = 100
	defaultSamplingRate = 1
	maximumSamplingRate = 10
	globalCounter       = "global"
)

// TransactionDetails - details about the transaction that are used for sampling
type TransactionDetails struct {
	Status string
	APIID  string
	SubID  string
}
