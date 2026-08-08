package domain

import (
	"errors"
	"testing"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim and lowercase", input: " User@Example.COM ", want: "user@example.com"},
		{name: "preserve provider alias", input: "User+Tag@Example.com", want: "user+tag@example.com"},
		{name: "deterministic normalized input", input: "user@example.com", want: "user@example.com"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := NormalizeEmail(test.input)
			if err != nil {
				t.Fatalf("normalize: %v", err)
			}
			if got.String() != test.want {
				t.Fatalf("got %q want %q", got.String(), test.want)
			}
			again, err := NormalizeEmail(got.String())
			if err != nil || again != got {
				t.Fatalf("normalization is not deterministic: again=%q err=%v", again, err)
			}
		})
	}
}

func TestNormalizeEmailRejectsInvalidDomainValues(t *testing.T) {
	for _, value := range []string{"", "   ", "missing-at", "@example.com", "user@", "two@@example.com", "user name@example.com"} {
		if _, err := NormalizeEmail(value); !errors.Is(err, ErrInvalidEmail) {
			t.Fatalf("expected invalid email for %q, got %v", value, err)
		}
	}
}
