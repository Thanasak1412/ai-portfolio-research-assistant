package runtime

import (
	"time"

	"github.com/google/uuid"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

type Clock struct{}

func (Clock) Now() time.Time { return time.Now() }

type IDGenerator struct{}

func (IDGenerator) UserID() (domain.UserID, error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return domain.UserID{}, err
	}
	return domain.NewUserID(value)
}

func (IDGenerator) SessionID() (domain.SessionID, error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return domain.SessionID{}, err
	}
	return domain.NewSessionID(value)
}

func (IDGenerator) TokenFamilyID() (domain.TokenFamilyID, error) {
	value, err := uuid.NewRandom()
	if err != nil {
		return domain.TokenFamilyID{}, err
	}
	return domain.NewTokenFamilyID(value)
}
