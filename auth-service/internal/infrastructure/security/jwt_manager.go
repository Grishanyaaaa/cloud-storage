package security

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
	audience        string
}

func NewJWTManager(cfg config.JWTConfig) (*JWTManager, error) {
	privBytes, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	pubBytes, err := os.ReadFile(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	return &JWTManager{
		privateKey:      privKey,
		publicKey:       pubKey,
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		issuer:          cfg.Issuer,
		audience:        cfg.Audience,
	}, nil
}

type accessClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"user_id"`
	Email  string `json:"email"`
}

func (m *JWTManager) GenerateAccessToken(claims port.TokenClaims, now time.Time) (string, error) {

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
		UserID: claims.UserID,
		Email:  claims.Email,
	})

	// подписываем приватным ключом  только auth-service может выпускать токены
	return token.SignedString(m.privateKey)
}

// GenerateRefreshToken просто рандомные байты, не JWT
// валидация через lookup хеша в базе
func (m *JWTManager) GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func (m *JWTManager) ParseAccessToken(tokenString string) (*port.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		// валидируем публичным ключом — любой сервис с публичным ключом может проверить токен
		return m.publicKey, nil
	},
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
	)
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return &port.TokenClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}

func (m *JWTManager) AccessTokenTTL() time.Duration {
	return m.accessTokenTTL
}

func (m *JWTManager) RefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}

func (m *JWTManager) GetJWKS() (interface{}, error) {
	// Для RS256 нам нужны модули (n) и экспонента (e) в base64-url кодировке
	n := m.publicKey.N
	e := m.publicKey.E

	// Формируем структуру JWKS согласно RFC 7517
	jwk := map[string]interface{}{
		"kty": "RSA",
		"use": "sig",
		"alg": "RS256",
		"kid": "main", // В идеале kid должен быть хешем ключа
		"n":   base64.RawURLEncoding.EncodeToString(n.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString([]byte{byte(e >> 16), byte(e >> 8), byte(e)}),
	}

	return map[string]interface{}{
		"keys": []interface{}{jwk},
	}, nil
}
