package application

import "errors"

var (
	ErrUserNotFound         = errors.New("identity user was not found")
	ErrDuplicateIdentity    = errors.New("identity already exists")
	ErrSessionNotFound      = errors.New("refresh session was not found")
	ErrPersistenceConflict  = errors.New("identity persistence conflict")
	ErrPersistenceRetryable = errors.New("identity persistence operation may be retried")
	ErrPersistenceFailure   = errors.New("identity persistence operation failed")
)

type PersistenceError struct {
	kind  error
	cause error
}

func NewPersistenceError(kind, cause error) error {
	return &PersistenceError{kind: kind, cause: cause}
}

func (err *PersistenceError) Error() string { return err.kind.Error() }
func (err *PersistenceError) Unwrap() error { return err.cause }
func (err *PersistenceError) Is(target error) bool {
	return target == err.kind || errors.Is(err.cause, target)
}
