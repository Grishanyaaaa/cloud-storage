package dto

type RegisterResponse struct {
	UserID string
}

type TokenPairResponse struct {
	AccessToken  string
	RefreshToken string
}
