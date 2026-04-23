package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/domain/valueobject"
)

// User represents a user entity in the system.
// This is a domain entity that contains business logic and invariants.
type User struct {
	id           valueobject.UserID
	email        valueobject.Email
	passwordHash string
	createdAt    time.Time
	updatedAt    time.Time
	lastLogin    *time.Time
	isActive     bool
}

// NewUser creates a new User entity with the provided values.
// This is the factory function for creating valid User entities.
func NewUser(
	id valueobject.UserID,
	email valueobject.Email,
	passwordHash string,
	createdAt time.Time,
) *User {
	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    createdAt,
		lastLogin:    nil,
		isActive:     true,
	}
}

// ReconstructUser reconstructs a User entity from persistence.
// Used when loading from database.
func ReconstructUser(
	id valueobject.UserID,
	email valueobject.Email,
	passwordHash string,
	createdAt time.Time,
	updatedAt time.Time,
	lastLogin *time.Time,
	isActive bool,
) *User {
	return &User{
		id:           id,
		email:        email,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
		lastLogin:    lastLogin,
		isActive:     isActive,
	}
}

// ID returns the user's unique identifier.
func (u *User) ID() valueobject.UserID {
	return u.id
}

// Email returns the user's email address.
func (u *User) Email() valueobject.Email {
	return u.email
}

// PasswordHash returns the user's password hash.
func (u *User) PasswordHash() string {
	return u.passwordHash
}

// CreatedAt returns the timestamp when the user was created.
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdatedAt returns the timestamp when the user was last updated.
func (u *User) UpdatedAt() time.Time {
	return u.updatedAt
}

// LastLogin returns the timestamp of the user's last login, or nil if never logged in.
func (u *User) LastLogin() *time.Time {
	return u.lastLogin
}

// IsActive returns whether the user account is active.
func (u *User) IsActive() bool {
	return u.isActive
}

// UpdateLastLogin updates the last login timestamp to the current time.
func (u *User) UpdateLastLogin() {
	now := time.Now()
	u.lastLogin = &now
	u.updatedAt = now
}

// SetPasswordHash updates the user's password hash.
func (u *User) SetPasswordHash(hash string) {
	u.passwordHash = hash
	u.updatedAt = time.Now()
}

// Deactivate deactivates the user account.
func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

// Activate activates the user account.
func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

// CanLogin checks if the user can perform login.
// Business rule: user must be active to login.
func (u *User) CanLogin() bool {
	return u.isActive
}

// VerifyPassword verifies if the provided password matches the stored hash.
// This method delegates to a PasswordService for actual verification.
func (u *User) VerifyPassword(password string, hashFunc func(string, string) (bool, error)) (bool, error) {
	return hashFunc(password, u.passwordHash)
}
