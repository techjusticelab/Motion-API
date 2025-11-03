package validation

import (
	"errors"
	"strings"
)

// ErrorSet accumulates validation failures keyed by field.
type ErrorSet struct {
	fields map[string][]string
}

// NewErrorSet initialises an error collector.
func NewErrorSet() *ErrorSet {
	return &ErrorSet{fields: make(map[string][]string)}
}

// Append records an error for the given field.
func (e *ErrorSet) Append(field string, err error) {
	if err == nil {
		return
	}
	if field == "" {
		field = "unknown"
	}
	e.fields[field] = append(e.fields[field], err.Error())
}

// Empty reports whether any errors have been recorded.
func (e *ErrorSet) Empty() bool {
	return len(e.fields) == 0
}

// Error converts the collected errors into a single error value.
func (e *ErrorSet) Error() error {
	if e == nil || e.Empty() {
		return nil
	}
	var parts []string
	for field, messages := range e.fields {
		parts = append(parts, field+": "+strings.Join(messages, ", "))
	}
	return errors.New(strings.Join(parts, "; "))
}
