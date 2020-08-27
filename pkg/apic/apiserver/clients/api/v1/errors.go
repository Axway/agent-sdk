package v1

import (
	"fmt"
	"strings"
	"text/template"

	apiv1 "git.ecd.axway.org/apigov/apic_agents_sdk/pkg/apic/apiserver/models/api/v1"
)

type Errors []apiv1.Error

func (e Errors) Error() string {
	b := &strings.Builder{}

	b.WriteRune('[')
	for i, err := range e {
		b.WriteString(fmt.Sprintf("{\"status\": %d, \"detail\": \"%s\"}", err.Status, template.JSEscapeString(err.Detail)))
		if i < len(e)-1 {
			b.WriteRune(',')
		}
	}
	b.WriteRune(']')

	return b.String()
}

type ConflictError struct {
	Errors
}

func (nf ConflictError) Error() string {
	return fmt.Sprintf("conflict: %s", nf.Errors)
}

type NotFoundError struct {
	Errors
}

func (nf NotFoundError) Error() string {
	return fmt.Sprintf("not found: %s", nf.Errors)
}

type InternalServerError struct {
	Errors
}

func (nf InternalServerError) Error() string {
	return fmt.Sprintf("internal server error: %s", nf.Errors)
}

type ForbiddenError struct {
	Errors
}

func (nf ForbiddenError) Error() string {
	return fmt.Sprintf("forbidden: %s", nf.Errors)
}

type UnauthorizedError struct {
	Errors
}

func (nf UnauthorizedError) Error() string {
	return fmt.Sprintf("unauthorized: %s", nf.Errors)
}

type BadRequestError struct {
	Errors
}

func (nf BadRequestError) Error() string {
	return fmt.Sprintf("bad request: %s", nf.Errors)
}

type UnexpectedError struct {
	code int
	Errors
}

func (nf UnexpectedError) Error() string {
	return fmt.Sprintf("unexpected code %d: %s", nf.code, nf.Errors)
}
