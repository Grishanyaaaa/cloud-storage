package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/auth-service/internal/infrastructure/config"
	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	privateKey      ed25519.PrivateKey
	publicKey       ed25519.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
	audience        string
}

func NewJWTManager(cfg config.JWTConfig) (*JWTManager, error) {
	if cfg.PrivateKey == "" {
		return nil, fmt.Errorf("private key is required (JWT_PRIVATE_KEY)")
	}

	// Декодируем seed из Base64
	seed, err := base64.StdEncoding.DecodeString(cfg.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("decode private key seed: %w", err)
	}
	if len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("invalid private key seed size: expected %d, got %d", ed25519.SeedSize, len(seed))
	}

	privKey := ed25519.NewKeyFromSeed(seed)

	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("public key is required (JWT_PUBLIC_KEY)")
	}

	// Декодируем публичный ключ из Base64
	pubKeyBytes, err := base64.StdEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: expected %d, got %d", ed25519.PublicKeySize, len(pubKeyBytes))
	}

	pubKey := ed25519.PublicKey(pubKeyBytes)

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

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
		UserID: claims.UserID,
		Email:  claims.Email,
	})

	// подписываем приватным ключом Ed25519
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
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
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
	// Для Ed25519 (EdDSA) формат JWK — OKP (Octet Key Pair)
	jwk := map[string]interface{}{
		"kty": "OKP",
		"use": "sig",
		"alg": "EdDSA",
		"crv": "Ed25519",
		"kid": "main",
		"x":   base64.RawURLEncoding.EncodeToString(m.publicKey),
	}

	return map[string]interface{}{
		"keys": []interface{}{jwk},
	}, nil
}
