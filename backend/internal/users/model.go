package users

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNotFound   = errors.New("profile not found")
	ErrConflict   = errors.New("profile version conflict")
	ErrValidation = errors.New("profile validation failed")
)

type Identity struct {
	Provider string
	Subject  string
}

type Consent struct {
	ID              uuid.UUID
	DocumentKey     string
	DocumentVersion string
	Source          string
	AcceptedAt      time.Time
	WithdrawnAt     *time.Time
}

type Profile struct {
	UserID      uuid.UUID
	DisplayName string
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Consents    []Consent
}

type Update struct {
	DisplayName     string
	ExpectedVersion int64
}

func NormalizeDisplayName(value string) (string, error) {
	value = strings.TrimSpace(value)
	length := utf8.RuneCountInString(value)
	if !utf8.ValidString(value) || length < 1 || length > 80 {
		return "", fmt.Errorf("%w: display name must contain 1 to 80 valid Unicode characters", ErrValidation)
	}
	for _, character := range value {
		if unicode.IsControl(character) {
			return "", fmt.Errorf("%w: display name cannot contain control characters", ErrValidation)
		}
	}
	return value, nil
}
