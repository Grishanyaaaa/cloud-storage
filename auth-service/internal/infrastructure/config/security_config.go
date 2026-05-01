package config

type SecurityConfig struct {
	BcryptCost        int  `env:"SECURITY_BCRYPT_COST" env-default:"12"`
	MinPasswordLength int  `env:"SECURITY_MIN_PASSWORD_LENGTH" env-default:"8"`
	MaxPasswordLength int  `env:"SECURITY_MAX_PASSWORD_LENGTH" env-default:"72"`
	TrustProxy        bool `env:"SECURITY_TRUST_PROXY" env-default:"false"`
}
