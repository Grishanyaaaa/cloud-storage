package valueobject

import (
	"errors"
	"testing"
)

func TestPasswordPolicy_NewPassword(t *testing.T) {
	policy := PasswordPolicy{
		MinLength: 8,
		MaxLength: 72,
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
		errType error
	}{
		{
			name:    "valid password",
			input:   "SecureP@ss123",
			wantErr: false,
		},
		{
			name:    "minimum valid password",
			input:   "Abcd123!",
			wantErr: false,
		},
		{
			name:    "too short",
			input:   "Ab1!",
			wantErr: true,
		},
		{
			name:    "too long",
			input:   "A1!" + string(make([]byte, 70)),
			wantErr: true,
		},
		{
			name:    "no uppercase",
			input:   "password123!",
			wantErr: true,
			errType: ErrPasswordNoUppercase,
		},
		{
			name:    "no lowercase",
			input:   "PASSWORD123!",
			wantErr: true,
			errType: ErrPasswordNoLowercase,
		},
		{
			name:    "no number",
			input:   "Password!",
			wantErr: true,
			errType: ErrPasswordNoNumber,
		},
		{
			name:    "no special character",
			input:   "Password123",
			wantErr: true,
			errType: ErrPasswordNoSpecial,
		},
		{
			name:    "multiple violations",
			input:   "password",
			wantErr: true,
		},
		{
			name:    "unicode special characters",
			input:   "SecureP@ss123",
			wantErr: false,
		},
		{
			name:    "symbols as special chars",
			input:   "SecureP$ss123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pwd, err := policy.NewPassword(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if pwd.String() != tt.input {
					t.Errorf("NewPassword() password value = %v, want %v", pwd.String(), tt.input)
				}
				if pwd.IsZero() {
					t.Errorf("NewPassword() password should not be zero")
				}
			}
			if tt.wantErr && tt.errType != nil {
				var validationErr PasswordValidationError
				if errors.As(err, &validationErr) {
					found := false
					for _, e := range validationErr.Errors {
						if errors.Is(e, tt.errType) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("NewPassword() error should contain %v, got %v", tt.errType, validationErr.Errors)
					}
				}
			}
		})
	}
}

func TestPasswordPolicy_ValidateRules(t *testing.T) {
	policy := PasswordPolicy{
		MinLength: 8,
		MaxLength: 72,
	}

	tests := []struct {
		name      string
		input     string
		wantErrs  int
		checkErrs []error
	}{
		{
			name:     "valid password returns no errors",
			input:    "SecureP@ss123",
			wantErrs: 0,
		},
		{
			name:     "empty password returns all errors",
			input:    "",
			wantErrs: 5, // too short + 4 character type errors
		},
		{
			name:      "missing uppercase only",
			input:     "password123!",
			wantErrs:  1,
			checkErrs: []error{ErrPasswordNoUppercase},
		},
		{
			name:      "missing lowercase only",
			input:     "PASSWORD123!",
			wantErrs:  1,
			checkErrs: []error{ErrPasswordNoLowercase},
		},
		{
			name:      "missing number only",
			input:     "Password!",
			wantErrs:  1,
			checkErrs: []error{ErrPasswordNoNumber},
		},
		{
			name:      "missing special only",
			input:     "Password123",
			wantErrs:  1,
			checkErrs: []error{ErrPasswordNoSpecial},
		},
		{
			name:     "all lowercase no special",
			input:    "password123",
			wantErrs: 2,
			checkErrs: []error{
				ErrPasswordNoUppercase,
				ErrPasswordNoSpecial,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := policy.ValidateRules(tt.input)
			if len(errs) != tt.wantErrs {
				t.Errorf("ValidateRules() returned %d errors, want %d. Errors: %v", len(errs), tt.wantErrs, errs)
			}
			for _, expectedErr := range tt.checkErrs {
				found := false
				for _, err := range errs {
					if errors.Is(err, expectedErr) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("ValidateRules() should contain error %v, got %v", expectedErr, errs)
				}
			}
		})
	}
}

func TestPasswordValidationError_Error(t *testing.T) {
	tests := []struct {
		name   string
		errors []error
		want   string
	}{
		{
			name:   "no errors",
			errors: []error{},
			want:   "password validation failed",
		},
		{
			name:   "single error",
			errors: []error{ErrPasswordNoUppercase},
			want:   "password must contain at least one uppercase letter",
		},
		{
			name: "multiple errors",
			errors: []error{
				ErrPasswordNoUppercase,
				ErrPasswordNoNumber,
			},
			want: "password validation failed: password must contain at least one uppercase letter; password must contain at least one number",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PasswordValidationError{Errors: tt.errors}
			if got := err.Error(); got != tt.want {
				t.Errorf("PasswordValidationError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPasswordValidationError_Unwrap(t *testing.T) {
	errs := []error{
		ErrPasswordNoUppercase,
		ErrPasswordNoNumber,
	}
	validationErr := PasswordValidationError{Errors: errs}

	unwrapped := validationErr.Unwrap()
	if len(unwrapped) != len(errs) {
		t.Errorf("Unwrap() returned %d errors, want %d", len(unwrapped), len(errs))
	}

	for i, err := range unwrapped {
		if !errors.Is(err, errs[i]) {
			t.Errorf("Unwrap()[%d] = %v, want %v", i, err, errs[i])
		}
	}
}

func TestPassword_IsZero(t *testing.T) {
	policy := PasswordPolicy{MinLength: 8, MaxLength: 72}

	t.Run("zero value password", func(t *testing.T) {
		pwd := Password{}
		if !pwd.IsZero() {
			t.Errorf("IsZero() = false, want true for zero value")
		}
	})

	t.Run("valid password is not zero", func(t *testing.T) {
		pwd, err := policy.NewPassword("SecureP@ss123")
		if err != nil {
			t.Fatalf("NewPassword() error = %v", err)
		}
		if pwd.IsZero() {
			t.Errorf("IsZero() = true, want false for valid password")
		}
	})
}

func TestPassword_String(t *testing.T) {
	policy := PasswordPolicy{MinLength: 8, MaxLength: 72}
	input := "SecureP@ss123"

	pwd, err := policy.NewPassword(input)
	if err != nil {
		t.Fatalf("NewPassword() error = %v", err)
	}

	if got := pwd.String(); got != input {
		t.Errorf("String() = %v, want %v", got, input)
	}
}

func TestPasswordPolicy_EdgeCases(t *testing.T) {
	policy := PasswordPolicy{
		MinLength: 8,
		MaxLength: 72,
	}

	t.Run("exactly min length", func(t *testing.T) {
		pwd := "Abcd123!"
		_, err := policy.NewPassword(pwd)
		if err != nil {
			t.Errorf("NewPassword() with exactly min length should succeed, got error: %v", err)
		}
	})

	t.Run("exactly max length", func(t *testing.T) {
		// Create a 72-char password
		pwd := "A1!" + string(make([]byte, 69))
		for i := 3; i < 72; i++ {
			pwd = pwd[:i] + "a" + pwd[i+1:]
		}
		_, err := policy.NewPassword(pwd)
		if err != nil {
			t.Errorf("NewPassword() with exactly max length should succeed, got error: %v", err)
		}
	})

	t.Run("unicode characters", func(t *testing.T) {
		pwd := "Пароль123!"
		_, err := policy.NewPassword(pwd)
		if err != nil {
			t.Errorf("NewPassword() with unicode should work, got error: %v", err)
		}
	})
}
