package client

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/api-gateway/internal/infrastructure/config"
	"github.com/golang-jwt/jwt/v5"
)

// JWK represents a JSON Web Key for Ed25519.
type JWK struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Crv string `json:"crv"`
	Kid string `json:"kid"`
	X   string `json:"x"`
}

// JWKS represents a JSON Web Key Set.
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWKSClient fetches and caches JWKS from auth-service.
type JWKSClient struct {
	jwksURL         string
	refreshInterval time.Duration
	httpClient      *http.Client

	mu         sync.RWMutex
	publicKeys map[string]ed25519.PublicKey
	lastFetch  time.Time
}

// NewJWKSClient creates a new JWKS client.
func NewJWKSClient(cfg config.JWTConfig) *JWKSClient {
	return &JWKSClient{
		jwksURL:         cfg.JWKSUrl,
		refreshInterval: cfg.RefreshInterval,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		publicKeys: make(map[string]ed25519.PublicKey),
	}
}

// FetchKeys fetches JWKS from auth-service and updates the cache.
func (c *JWKSClient) FetchKeys(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create jwks request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("jwks endpoint returned status %d", resp.StatusCode)
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	// Parse Ed25519 public keys
	keys := make(map[string]ed25519.PublicKey)
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "OKP" || jwk.Crv != "Ed25519" {
			continue
		}

		pubKeyBytes, err := base64.RawURLEncoding.DecodeString(jwk.X)
		if err != nil {
			return fmt.Errorf("decode public key for kid %s: %w", jwk.Kid, err)
		}

		if len(pubKeyBytes) != ed25519.PublicKeySize {
			return fmt.Errorf("invalid public key size for kid %s", jwk.Kid)
		}

		keys[jwk.Kid] = ed25519.PublicKey(pubKeyBytes)
	}

	if len(keys) == 0 {
		return fmt.Errorf("no valid Ed25519 keys found in JWKS")
	}

	c.mu.Lock()
	c.publicKeys = keys
	c.lastFetch = time.Now()
	c.mu.Unlock()

	return nil
}

// GetPublicKey returns the public key for the given kid.
func (c *JWKSClient) GetPublicKey(kid string) (ed25519.PublicKey, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key, ok := c.publicKeys[kid]
	if !ok {
		return nil, fmt.Errorf("public key not found for kid: %s", kid)
	}

	return key, nil
}

// ShouldRefresh checks if keys should be refreshed.
func (c *JWKSClient) ShouldRefresh() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return time.Since(c.lastFetch) > c.refreshInterval
}

// StartBackgroundRefresh starts a background goroutine to refresh keys periodically.
func (c *JWKSClient) StartBackgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.FetchKeys(ctx); err != nil {
				// Log error but don't stop the refresh loop
				fmt.Printf("failed to refresh JWKS: %v\n", err)
			}
		case <-ctx.Done():
			return
		}
	}
}

// ValidateToken validates a JWT token using the cached public keys.
func (c *JWKSClient) ValidateToken(tokenString string, issuer, audience string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get kid from header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			kid = "main" // default kid
		}

		// Get public key
		pubKey, err := c.GetPublicKey(kid)
		if err != nil {
			return nil, err
		}

		return pubKey, nil
	},
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
	)

	if err != nil {
		return nil, fmt.Errorf("validate token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token, nil
}
