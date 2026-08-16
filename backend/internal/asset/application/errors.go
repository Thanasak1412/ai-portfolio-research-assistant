package application

import "errors"

var (
	ErrUnauthenticated    = errors.New("authenticated principal is required")
	ErrInvalidAssetInput  = errors.New("invalid asset input")
	ErrAssetNotFound      = errors.New("asset was not found")
	ErrAssetService       = errors.New("asset service unavailable")
	ErrPersistenceFailure = errors.New("asset persistence operation failed")
)

type PersistenceError struct{ kind, cause error }

func NewPersistenceError(kind, cause error) error { return &PersistenceError{kind: kind, cause: cause} }
func (err *PersistenceError) Error() string       { return err.kind.Error() }
func (err *PersistenceError) Unwrap() error       { return err.cause }
func (err *PersistenceError) Is(target error) bool {
	return target == err.kind || errors.Is(err.cause, target)
}
