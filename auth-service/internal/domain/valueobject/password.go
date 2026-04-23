package valueobject

import (
	"errors"
	"unicode"
)

// Password validation errors.
var (
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong     = errors.New("password must not exceed 72 characters")
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber    = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial   = errors.New("password must contain at least one special character")
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72 // bcrypt limit
)

// Password represents a plain-text password for validation.
// This is a value object used during registration/login.
// It should never be stored - only used for validation and hashing.
type Password struct {
	value string
}

// NewPassword creates a new Password value object with validation.
// Returns an error if the password doesn't meet security requirements.
func NewPassword(password string) (Password, error) {
	// Check length
	if len(password) < minPasswordLength {
		return Password{}, ErrPasswordTooShort
	}
	if len(password) > maxPasswordLength {
		return Password{}, ErrPasswordTooLong
	}

	// Validate complexity
	var (
		hasUpper   bool
		hasLower   bool
		hasNumber  bool
		hasSpecial bool
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return Password{}, ErrPasswordNoUppercase
	}
	if !hasLower {
		return Password{}, ErrPasswordNoLowercase
	}
	if !hasNumber {
		return Password{}, ErrPasswordNoNumber
	}
	if !hasSpecial {
		return Password{}, ErrPasswordNoSpecial
	}

	return Password{value: password}, nil
}

// String returns the plain-text password.
// WARNING: This should only be used for hashing, never for storage or logging.
func (p Password) String() string {
	return p.value
}

// IsZero returns true if the Password is the zero value.
func (p Password) IsZero() bool {
	return p.value == ""
}

// ValidatePasswordRules returns all validation errors for a password.
// Useful for providing detailed feedback to users.
func ValidatePasswordRules(password string) []error {
	var errs []error

	if len(password) < minPasswordLength {
		errs = append(errs, ErrPasswordTooShort)
	}
	if len(password) > maxPasswordLength {
		errs = append(errs, ErrPasswordTooLong)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		errs = append(errs, ErrPasswordNoUppercase)
	}
	if !hasLower {
		errs = append(errs, ErrPasswordNoLowercase)
	}
	if !hasNumber {
		errs = append(errs, ErrPasswordNoNumber)
	}
	if !hasSpecial {
		errs = append(errs, ErrPasswordNoSpecial)
	}

	return errs
}
