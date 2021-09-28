package transaction

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTransactionEventStatus(t *testing.T) {
	assert.Equal(t, "Fail", GetTransactionEventStatus(401))
	assert.Equal(t, "Fail", GetTransactionEventStatus(500))
	assert.Equal(t, "Pass", GetTransactionEventStatus(201))
}

func TestGetTransactionSummaryStatus(t *testing.T) {
	assert.Equal(t, "Success", GetTransactionSummaryStatus(201))
	assert.Equal(t, "Failure", GetTransactionSummaryStatus(404))
	assert.Equal(t, "Exception", GetTransactionSummaryStatus(501))
	assert.Equal(t, "Unknown", GetTransactionSummaryStatus(555))
}

func TestMarshalHeadersAsJSONString(t *testing.T) {
	m := map[string]string{}
	assert.Equal(t, "{}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "val1",
		"prop2": "val2",
	}
	assert.Equal(t, "{\"prop1\":\"val1\",\"prop2\":\"val2\"}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "val1",
		"prop2": "",
	}
	assert.Equal(t, "{\"prop1\":\"val1\",\"prop2\":\"\"}", MarshalHeadersAsJSONString(m))

	m = map[string]string{
		"prop1": "aaa\"bbb\"ccc",
	}
	assert.Equal(t, "{\"prop1\":\"aaa\\\"bbb\\\"ccc\"}", MarshalHeadersAsJSONString(m))
}

func TestFormatProxyID(t *testing.T) {
	s := FormatProxyID("foobar")
	assert.Equal(t, SummaryEventProxyIDPrefix+"foobar", s)
}
func TestFormatApplicationID(t *testing.T) {
	s := FormatApplicationID("barfoo")
	assert.Equal(t, SummaryEventApplicationIDPrefix+"barfoo", s)
}
