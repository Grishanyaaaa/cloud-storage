package valueobject

import (
	"errors"
	"regexp"
	"strings"
)

// ErrInvalidEmail is returned when an email is invalid.
var ErrInvalidEmail = errors.New("invalid email format")

// ErrEmailTooLong is returned when an email exceeds the maximum length.
var ErrEmailTooLong = errors.New("email exceeds maximum length")

const maxEmailLength = 255

// Email regex pattern - validates common email formats.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// Email represents an email address value object.
// It enforces validation invariants at creation time.
type Email struct {
	value string
}

// NewEmail creates a new Email value object.
// Returns an error if the email is invalid.
func NewEmail(email string) (Email, error) {
	// Normalize email
	email = strings.TrimSpace(strings.ToLower(email))

	// Validate length
	if len(email) > maxEmailLength {
		return Email{}, ErrEmailTooLong
	}

	// Validate format
	if !emailRegex.MatchString(email) {
		return Email{}, ErrInvalidEmail
	}

	return Email{value: email}, nil
}

// String returns the email address as a string.
func (e Email) String() string {
	return e.value
}

// Equals checks if two Email values are equal.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// IsZero returns true if the Email is the zero value.
func (e Email) IsZero() bool {
	return e.value == ""
}
