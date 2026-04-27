package port

import "time"

type TokenClaims struct {
	UserID string
	Email  string
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string // сырой токен, хешировать перед записью в базу
}

type TokenManager interface {
	GenerateAccessToken(claims TokenClaims, now time.Time) (string, error)
	GenerateRefreshToken() (string, error)
	ParseAccessToken(token string) (*TokenClaims, error)
	AccessTokenTTL() time.Duration
	RefreshTokenTTL() time.Duration
	GetJWKS() (interface{}, error)
}
