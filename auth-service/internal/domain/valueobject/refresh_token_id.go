package valueobject

import "github.com/google/uuid"

type RefreshTokenID string

func NewRefreshTokenID() RefreshTokenID {
	return RefreshTokenID(uuid.New().String())
}

func (id RefreshTokenID) String() string {
	return string(id)
}
