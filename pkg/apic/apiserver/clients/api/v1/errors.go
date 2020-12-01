package v1

import (
	"fmt"
	"strings"

	apiv1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

// Errors -
type Errors []apiv1.Error

// Error -
func (e Errors) Error() string {
	b := &strings.Builder{}

	b.WriteRune('[')
	for i, err := range e {
		b.WriteString(fmt.Sprintf("{\"status\": %d, \"detail\": \"%s\"}", err.Status, err.Detail))
		if i < len(e)-1 {
			b.WriteRune(',')
		}
	}
	b.WriteRune(']')

	return b.String()
}

// ConflictError -
type ConflictError struct {
	Errors
}

// Error -
func (nf ConflictError) Error() string {
	return fmt.Sprintf("conflict: %s", nf.Errors)
}

// NotFoundError -
type NotFoundError struct {
	Errors
}

// Error -
func (nf NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", nf.Errors)
}

// InternalServerError -
type InternalServerError struct {
	Errors
}

// Error -
func (nf InternalServerError) Error() string {
	return fmt.Sprintf("internal server error: %s", nf.Errors)
}

// ForbiddenError -
type ForbiddenError struct {
	Errors
}

// Error -
func (nf ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden: %s", nf.Errors)
}

// UnauthorizedError -
type UnauthorizedError struct {
	Errors
}

// Error -
func (nf UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", nf.Errors)
}

// BadRequestError -
type BadRequestError struct {
	Errors
}

// Error -
func (nf BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", nf.Errors)
}

// UnexpectedError -
type UnexpectedError struct {
	code int
	Errors
}

// Error -
func (nf UnexpectedError) Error() string {
	return fmt.Sprintf("unexpected code %d: %s", nf.code, nf.Errors)
}
