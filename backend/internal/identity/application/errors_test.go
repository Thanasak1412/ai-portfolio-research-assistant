package application

import (
	"errors"
	"strings"
	"testing"
)

func TestPersistenceErrorClassifiesWithoutLeakingCauseText(t *testing.T) {
	sensitive := errors.New("database rejected secret@example.test and credential-material")
	err := NewPersistenceError(ErrPersistenceConflict, sensitive)
	if !errors.Is(err, ErrPersistenceConflict) || !errors.Is(err, sensitive) {
		t.Fatal("persistence error did not preserve classification and internal cause")
	}
	if strings.Contains(err.Error(), "secret@example.test") || strings.Contains(err.Error(), "credential-material") {
		t.Fatalf("persistence error leaked sensitive cause: %q", err.Error())
	}
}
