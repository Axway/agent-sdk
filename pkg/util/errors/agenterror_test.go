package errors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewError(t *testing.T) {
	code := 1001
	msg := "this is a test error"
	newErr := New(code, msg)

	assert.NotNil(t, newErr, "The error returned by New was nil")
	assert.IsType(t, &AgentError{}, newErr, "The new error was not of AgentError type")
	assert.Implements(t, (*error)(nil), newErr, "The AgentError struct does not implement error")
	assert.Contains(t, newErr.Error(), msg, "The error msg returned was incorrect")
	assert.Contains(t, newErr.FormatError().Error(), msg, "The error msg returned was incorrect")
	assert.Equal(t, code, newErr.GetErrorCode(), "The error code returned was incorrect")
}

func TestNewfError(t *testing.T) {
	code := 1001
	msg := "format %s test error"
	newErr := Newf(code, msg)

	assert.NotNil(t, newErr, "The error returned by New was nil")
	assert.IsType(t, &AgentError{}, newErr, "The new error was not of AgentError type")
	assert.Implements(t, (*error)(nil), newErr, "The AgentError struct does not implement error")
	assert.Contains(t, newErr.FormatError("value").Error(), fmt.Sprintf(msg, "value"), "The error msg returned was incorrect")
	assert.Equal(t, code, newErr.GetErrorCode(), "The error code returned was incorrect")
}

func TestWrapError(t *testing.T) {
	code := 1001
	msg := "this is a test error"
	newErr := New(code, msg)

	wrapMsg := "wrapped message"
	wrapErr := Wrap(newErr, wrapMsg)

	assert.NotNil(t, wrapErr, "The error returned by Wrap was nil")
	assert.IsType(t, &AgentError{}, wrapErr, "The new error was not of AgentError type")
	assert.Implements(t, (*error)(nil), wrapErr, "The AgentError struct does not implement error")
	assert.Contains(t, wrapErr.Error(), msg+": "+wrapMsg, "The error msg returned was incorrect")
	assert.Contains(t, wrapErr.FormatError().Error(), msg+": "+wrapMsg, "The error msg returned was incorrect")
	assert.Equal(t, code, wrapErr.GetErrorCode(), "The error code returned was incorrect")
}
