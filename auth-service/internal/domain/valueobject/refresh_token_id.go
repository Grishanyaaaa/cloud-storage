package valueobject

import (
	"fmt"

	"github.com/google/uuid"
)

type RefreshTokenID string

func NewRefreshTokenID() RefreshTokenID {
	return RefreshTokenID(uuid.New().String())
}

func (id RefreshTokenID) String() string {
	return string(id)
}

func ParseRefreshTokenID(raw string) (RefreshTokenID, error) {
	if _, err := uuid.Parse(raw); err != nil {
		return "", fmt.Errorf("invalid refresh token id: %w", err)
	}
	return RefreshTokenID(raw), nil
}
