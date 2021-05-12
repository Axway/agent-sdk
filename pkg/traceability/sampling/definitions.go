package sampling

//SampleKey - the key used in the metadata when a transaction qualifies for sampling and should be sent to Observer
const SampleKey = "sample"

//TransactionDetails - details about the transaction that are used for sampling
type TransactionDetails struct {
	Status string
}
