package valueobject

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// Password validation errors.
var (
	ErrPasswordNoUppercase = errors.New("password must contain at least one uppercase letter")
	ErrPasswordNoLowercase = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNoNumber    = errors.New("password must contain at least one number")
	ErrPasswordNoSpecial   = errors.New("password must contain at least one special character")
)

// PasswordValidationError contains multiple validation errors.
type PasswordValidationError struct {
	Errors []error
}

func (e PasswordValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "password validation failed"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	var sb strings.Builder
	sb.WriteString("password validation failed: ")
	for i, err := range e.Errors {
		if i > 0 {
			sb.WriteString("; ")
		}
		sb.WriteString(err.Error())
	}
	return sb.String()
}

func (e PasswordValidationError) Unwrap() []error {
	return e.Errors
}

// PasswordPolicy defines the rules for password validation.
// Created from configuration at startup, injected into use cases.
type PasswordPolicy struct {
	MinLength int
	MaxLength int
}

// Password represents a plain-text password for validation.
// This is a value object used during registration/login.
// It should never be stored — only used for validation and hashing.
type Password struct {
	value string
}

// NewPassword creates a new Password value object with validation against the policy.
// Returns all validation errors if the password doesn't meet security requirements.
func (p PasswordPolicy) NewPassword(password string) (Password, error) {
	if errs := p.ValidateRules(password); len(errs) > 0 {
		return Password{}, PasswordValidationError{Errors: errs}
	}
	return Password{value: password}, nil
}

// ValidateRules returns all validation errors for a password.
// Useful for providing detailed feedback to users.
func (p PasswordPolicy) ValidateRules(password string) []error {
	var errs []error

	if len(password) < p.MinLength {
		errs = append(errs, fmt.Errorf("password must be at least %d characters", p.MinLength))
	}
	if len(password) > p.MaxLength {
		errs = append(errs, fmt.Errorf("password must not exceed %d characters", p.MaxLength))
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
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

// String returns the plain-text password.
// WARNING: This should only be used for hashing, never for storage or logging.
func (p Password) String() string {
	return p.value
}

// IsZero returns true if the Password is the zero value.
func (p Password) IsZero() bool {
	return p.value == ""
}
