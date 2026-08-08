package domain

import (
	"fmt"
	"time"
)

type AccountStatus string

const (
	AccountStatusActive   AccountStatus = "active"
	AccountStatusDisabled AccountStatus = "disabled"
)

func ParseAccountStatus(value string) (AccountStatus, error) {
	status := AccountStatus(value)
	if status != AccountStatusActive && status != AccountStatusDisabled {
		return "", ErrInvalidAccountStatus
	}
	return status, nil
}

type User struct {
	id            UserID
	email         NormalizedEmail
	passwordHash  PasswordHash
	status        AccountStatus
	createdAt     time.Time
	updatedAt     time.Time
	disabledAt    time.Time
	hasDisabledAt bool
}

func NewActiveUser(id UserID, email NormalizedEmail, passwordHash PasswordHash, createdAt time.Time) (User, error) {
	return NewUser(id, email, passwordHash, AccountStatusActive, createdAt, createdAt, nil)
}

func NewUser(
	id UserID,
	email NormalizedEmail,
	passwordHash PasswordHash,
	status AccountStatus,
	createdAt time.Time,
	updatedAt time.Time,
	disabledAt *time.Time,
) (User, error) {
	if id.IsZero() || email.IsZero() || passwordHash.IsZero() || createdAt.IsZero() || updatedAt.Before(createdAt) {
		return User{}, ErrInvalidUser
	}
	if _, err := ParseAccountStatus(string(status)); err != nil {
		return User{}, err
	}
	if status == AccountStatusActive && disabledAt != nil {
		return User{}, ErrInvalidUser
	}
	if status == AccountStatusDisabled && (disabledAt == nil || disabledAt.Before(createdAt)) {
		return User{}, ErrInvalidUser
	}
	user := User{
		id: id, email: email, passwordHash: passwordHash, status: status,
		createdAt: createdAt, updatedAt: updatedAt,
	}
	if disabledAt != nil {
		user.disabledAt = *disabledAt
		user.hasDisabledAt = true
	}
	return user, nil
}

func (user User) ID() UserID                    { return user.id }
func (user User) Email() NormalizedEmail        { return user.email }
func (user User) PasswordHash() PasswordHash    { return user.passwordHash }
func (user User) Status() AccountStatus         { return user.status }
func (user User) CreatedAt() time.Time          { return user.createdAt }
func (user User) UpdatedAt() time.Time          { return user.updatedAt }
func (user User) IsActive() bool                { return user.status == AccountStatusActive }
func (user User) IsDisabled() bool              { return user.status == AccountStatusDisabled }
func (user User) DisabledAt() (time.Time, bool) { return user.disabledAt, user.hasDisabledAt }
func (user User) String() string                { return fmt.Sprintf("User{id:%s,status:%s}", user.id, user.status) }
func (user User) GoString() string              { return user.String() }
