package apperrors

import (
	"fmt"
)

type NotFoundError struct {
	Resource   string
	Identifier string
}

func (e *NotFoundError) Error() string {
	if e == nil {
		return "resource not found"
	}
	if e.Resource == "" && e.Identifier == "" {
		return "resource not found"
	}
	if e.Identifier == "" {
		return fmt.Sprintf("%s not found", e.Resource)
	}
	if e.Resource == "" {
		return fmt.Sprintf("resource %s not found", e.Identifier)
	}
	return fmt.Sprintf("%s %s not found", e.Resource, e.Identifier)
}

// Is allows errors.Is(err, &NotFoundError{}).
func (e *NotFoundError) Is(target error) bool {
	_, ok := target.(*NotFoundError)
	return ok
}

func NewNotFound(resource, identifier string) error {
	return &NotFoundError{Resource: resource, Identifier: identifier}
}

type NoMetricsError struct {
	Scope string
}

func (e *NoMetricsError) Error() string {
	if e == nil || e.Scope == "" {
		return "no metrics available"
	}
	return fmt.Sprintf("no metrics available for %s", e.Scope)
}

// Is allows errors.Is(err, &NoMetricsError{}).
func (e *NoMetricsError) Is(target error) bool {
	_, ok := target.(*NoMetricsError)
	return ok
}

func NewNoMetrics(scope string) error {
	return &NoMetricsError{Scope: scope}
}
