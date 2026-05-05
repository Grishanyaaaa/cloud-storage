package config

// JWTConfig configures JWT verification.
// ai-service only verifies access tokens issued by auth-service —
// it never signs them. We only need the public key.
type JWTConfig struct {
	PublicKey string `env:"JWT_PUBLIC_KEY" env-required:"true"` // Base64 encoded 32-byte ed25519 public key
	Issuer    string `env:"JWT_ISSUER" env-default:"auth-service"`
	Audience  string `env:"JWT_AUDIENCE" env-default:"cloud-storage"`
}
