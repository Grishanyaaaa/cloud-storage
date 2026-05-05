package security

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/infrastructure/config"
)

// Compile-time check
var _ port.JWTParser = (*JWTParser)(nil)

// JWTParser validates Ed25519 JWTs issued by auth-service.
// ai-service NEVER signs tokens — only verifies and reads payload.
//
// In addition to extracting the claims, it preserves the verbatim token in
// JWTPayload.RawToken so the use-case layer can propagate it to storage-service.
type JWTParser struct {
	publicKey ed25519.PublicKey
	issuer    string
	audience  string
}

// NewJWTParser builds a parser from a base64-encoded 32-byte ed25519 public key.
func NewJWTParser(cfg config.JWTConfig) (*JWTParser, error) {
	if cfg.PublicKey == "" {
		return nil, fmt.Errorf("public key is required (JWT_PUBLIC_KEY)")
	}
	pubKeyBytes, err := base64.StdEncoding.DecodeString(cfg.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(pubKeyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf(
			"invalid public key size: expected %d, got %d",
			ed25519.PublicKeySize, len(pubKeyBytes),
		)
	}
	return &JWTParser{
		publicKey: ed25519.PublicKey(pubKeyBytes),
		issuer:    cfg.Issuer,
		audience:  cfg.Audience,
	}, nil
}

// accessClaims mirrors the access-token shape produced by auth-service.
type accessClaims struct {
	jwt.RegisteredClaims
	UserID string   `json:"user_id"`
	Email  string   `json:"email"`
	Roles  []string `json:"roles,omitempty"`
}

// Parse extracts a Bearer token from the Authorization header and validates it.
// Returns:
//   - ErrUnauthorized   when header is missing or malformed
//   - ErrTokenExpired   when the token is past its exp
//   - ErrInvalidToken   on any other validation failure
func (p *JWTParser) Parse(r *http.Request) (*port.JWTPayload, error) {
	header := r.Header.Get("Authorization")
	if header == "" {
		return nil, domainerr.ErrUnauthorized
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return nil, domainerr.ErrUnauthorized
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	if tokenString == "" {
		return nil, domainerr.ErrUnauthorized
	}

	token, err := jwt.ParseWithClaims(tokenString, &accessClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return p.publicKey, nil
	},
		jwt.WithIssuer(p.issuer),
		jwt.WithAudience(p.audience),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domainerr.ErrTokenExpired
		}
		return nil, domainerr.ErrInvalidToken
	}

	claims, ok := token.Claims.(*accessClaims)
	if !ok || !token.Valid {
		return nil, domainerr.ErrInvalidToken
	}
	if claims.UserID == "" {
		return nil, domainerr.ErrInvalidToken
	}
	return &port.JWTPayload{
		UserID:   claims.UserID,
		Email:    claims.Email,
		Roles:    append([]string(nil), claims.Roles...),
		RawToken: tokenString,
	}, nil
}
