package dto

type RegisterRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type LoginRequest struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	IPAddress string `json:"-"`
	UserAgent string `json:"-"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
	IPAddress    string `json:"-"`
	UserAgent    string `json:"-"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
	IPAddress    string `json:"-"`
	UserAgent    string `json:"-"`
}

// Validate validates RegisterRequest fields.
func (r *RegisterRequest) Validate() error {
	if r.Email == "" {
		return ErrEmailRequired
	}
	if r.Password == "" {
		return ErrPasswordRequired
	}
	// bcrypt has a maximum password length of 72 bytes
	if len(r.Password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

// Validate validates LoginRequest fields.
func (r *LoginRequest) Validate() error {
	if r.Email == "" {
		return ErrEmailRequired
	}
	if r.Password == "" {
		return ErrPasswordRequired
	}
	// bcrypt has a maximum password length of 72 bytes
	if len(r.Password) > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

// Validate validates RefreshRequest fields.
func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	// Refresh token должен быть 64 символа (32 байта в hex)
	if len(r.RefreshToken) != 64 {
		return ErrInvalidRefreshTokenFormat
	}
	// Проверка, что это валидный hex
	for _, c := range r.RefreshToken {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return ErrInvalidRefreshTokenFormat
		}
	}
	return nil
}

// Validate validates LogoutRequest fields.
func (r *LogoutRequest) Validate() error {
	if r.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	// Refresh token должен быть 64 символа (32 байта в hex)
	if len(r.RefreshToken) != 64 {
		return ErrInvalidRefreshTokenFormat
	}
	// Проверка, что это валидный hex
	for _, c := range r.RefreshToken {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return ErrInvalidRefreshTokenFormat
		}
	}
	return nil
}
