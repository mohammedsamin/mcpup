package core

import "fmt"

const (
	ExitCodeRuntime          = 1
	ExitCodeValidation       = 2
	ExitCodePartialRecovered = 3
)

// ExitError allows callers to return stable process exit codes.
type ExitError interface {
	error
	ExitCode() int
}

// ReconcileError represents reconciliation failures with explicit exit codes.
type ReconcileError struct {
	Code int
	Err  error
}

func (e *ReconcileError) Error() string {
	return e.Err.Error()
}

func (e *ReconcileError) Unwrap() error {
	return e.Err
}

// ExitCode returns process code for this failure.
func (e *ReconcileError) ExitCode() int {
	return e.Code
}

func newReconcileError(code int, format string, args ...any) error {
	return &ReconcileError{
		Code: code,
		Err:  fmt.Errorf(format, args...),
	}
}
