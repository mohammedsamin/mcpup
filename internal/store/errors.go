package store

import "fmt"

// ErrorKind identifies the category of a store error.
type ErrorKind string

const (
	KindNotFound   ErrorKind = "not_found"
	KindDecode     ErrorKind = "decode"
	KindValidation ErrorKind = "validation"
	KindIO         ErrorKind = "io"
)

// StoreError is a typed error returned by store load/save operations.
type StoreError struct {
	Op   string
	Path string
	Kind ErrorKind
	Err  error
}

func (e *StoreError) Error() string {
	return fmt.Sprintf("%s %s (%s): %v", e.Op, e.Path, e.Kind, e.Err)
}

// Unwrap returns the underlying error.
func (e *StoreError) Unwrap() error {
	return e.Err
}

func newStoreError(op string, path string, kind ErrorKind, err error) error {
	return &StoreError{
		Op:   op,
		Path: path,
		Kind: kind,
		Err:  err,
	}
}
