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
	return nil
}

// Validate validates RefreshRequest fields.
func (r *RefreshRequest) Validate() error {
	if r.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	return nil
}

// Validate validates LogoutRequest fields.
func (r *LogoutRequest) Validate() error {
	if r.RefreshToken == "" {
		return ErrRefreshTokenRequired
	}
	return nil
}
