package apic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var methods = [5]string{"get", "post", "put", "patch", "delete"} // RestAPI methods

func TestSanitizeAPIName(t *testing.T) {
	name := sanitizeAPIName("Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc.Def")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName(".Abc...De/f")
	assert.Equal(t, "abc--.de-f", name)
	name = sanitizeAPIName("Abc.D-ef")
	assert.Equal(t, "abc.d-ef", name)
	name = sanitizeAPIName("Abc.Def=")
	assert.Equal(t, "abc.def", name)
	name = sanitizeAPIName("A..bc.Def")
	assert.Equal(t, "a--bc.def", name)
}
