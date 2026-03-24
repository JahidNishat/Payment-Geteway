package errors

import "fmt"

type NotFoundError struct {
	Resource string
	ID       any
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with id %v not found", e.Resource, e.ID)
}

// Validation Error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// Conflict Error
type ConflictError struct {
	Resource string
	Field    string
	Value    string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%s with %s '%s' already exists",
		e.Resource, e.Field, e.Value,
	)
}
