package errors

import (
	"fmt"
)

// AgentError - Agent Error
type AgentError struct {
	error

	formattedErr bool
	Code         int    `json:"code" structs:"code,omitempty"`
	Message      string `json:"message" structs:"message,omitempty"`
	formatArgs   []interface{}
}

// New - Creates a new Agent Error
func New(errCode int, errMessage string) *AgentError {
	return &AgentError{formattedErr: false, Code: errCode, Message: errMessage}
}

// Newf - Creates a new formatted Agent Error
func Newf(errCode int, errMessage string) *AgentError {
	return &AgentError{formattedErr: true, Code: errCode, Message: errMessage}
}

// Wrap -add additional data to a defined error
func Wrap(agentError *AgentError, info string) *AgentError {
	message := agentError.Message
	if info != "" {
		message += fmt.Sprintf(": %s", info)
	}
	return &AgentError{
		formattedErr: agentError.formattedErr,
		Code:         agentError.Code,
		Message:      message,
		formatArgs:   agentError.formatArgs,
	}
}

// FormatError - Creates a Error with applied formatting
func (e *AgentError) FormatError(args ...interface{}) error {
	return &AgentError{formattedErr: e.formattedErr, Code: e.Code, Message: e.Message, formatArgs: args}
}

// Error - Returns the formatted error message
func (e *AgentError) Error() string {
	if e.formattedErr {
		formattedMsg := fmt.Sprintf(e.Message, e.formatArgs...)
		return fmt.Sprintf("[Error Code %d] - %s", e.Code, formattedMsg)
	}

	return fmt.Sprintf("[Error Code %d] - %s", e.Code, e.Message)
}

// GetErrorCode - Returns the error code
func (e *AgentError) GetErrorCode() int {
	return e.Code
}
