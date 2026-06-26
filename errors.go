package courier

import "fmt"

// ErrorKind classifies failures in a provider-neutral way.
type ErrorKind string

const (
	ErrorKindConfiguration  ErrorKind = "configuration"
	ErrorKindValidation     ErrorKind = "validation"
	ErrorKindAuthentication ErrorKind = "authentication"
	ErrorKindNotFound       ErrorKind = "not_found"
	ErrorKindAPI            ErrorKind = "api"
	ErrorKindNetwork        ErrorKind = "network"
)

// Error is the shared error type returned by courier implementations.
type Error struct {
	Kind       ErrorKind
	Message    string
	Field      string
	StatusCode int
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Field != "" {
		return fmt.Sprintf("%s error for %q: %s", e.Kind, e.Field, e.Message)
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("%s error (%d): %s", e.Kind, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// IsKind reports whether err is a courier Error of the given kind.
func IsKind(err error, kind ErrorKind) bool {
	if err == nil {
		return false
	}
	cerr, ok := err.(*Error)
	return ok && cerr.Kind == kind
}

func configurationError(message string) *Error {
	return &Error{Kind: ErrorKindConfiguration, Message: message}
}

func validationError(field, message string) *Error {
	return &Error{Kind: ErrorKindValidation, Field: field, Message: message}
}
