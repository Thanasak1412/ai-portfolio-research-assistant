package domain

import "strings"

type NormalizedEmail struct{ value string }

func NormalizeEmail(value string) (NormalizedEmail, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" || strings.ContainsAny(normalized, "\r\n\t ") {
		return NormalizedEmail{}, ErrInvalidEmail
	}
	local, emailDomain, found := strings.Cut(normalized, "@")
	if !found || local == "" || emailDomain == "" || strings.Contains(emailDomain, "@") {
		return NormalizedEmail{}, ErrInvalidEmail
	}
	return NormalizedEmail{value: normalized}, nil
}

func (email NormalizedEmail) String() string { return email.value }
func (email NormalizedEmail) IsZero() bool   { return email.value == "" }
