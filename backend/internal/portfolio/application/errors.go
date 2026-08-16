package application

import "errors"

var (
	ErrUnauthenticated       = errors.New("authenticated principal is required")
	ErrInvalidPortfolioInput = errors.New("invalid portfolio input")
	ErrPortfolioNotFound     = errors.New("portfolio was not found")
	ErrPortfolioNameConflict = errors.New("portfolio name conflicts with an active portfolio")
	ErrPortfolioArchived     = errors.New("portfolio is archived")
	ErrPersistenceConflict   = errors.New("portfolio persistence conflict")
	ErrPersistenceRetryable  = errors.New("portfolio persistence operation may be retried")
	ErrPersistenceFailure    = errors.New("portfolio persistence operation failed")
	ErrPortfolioService      = errors.New("portfolio service unavailable")
)

type PersistenceError struct {
	kind  error
	cause error
}

func NewPersistenceError(kind, cause error) error { return &PersistenceError{kind: kind, cause: cause} }
func (err *PersistenceError) Error() string       { return err.kind.Error() }
func (err *PersistenceError) Unwrap() error       { return err.cause }
func (err *PersistenceError) Is(target error) bool {
	return target == err.kind || errors.Is(err.cause, target)
}
