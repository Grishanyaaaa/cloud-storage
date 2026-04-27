package dto

type RegisterRequest struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type LoginRequest struct {
	Email     string
	Password  string
	IPAddress string
	UserAgent string
}

type RefreshRequest struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}

type LogoutRequest struct {
	RefreshToken string
	IPAddress    string
	UserAgent    string
}
