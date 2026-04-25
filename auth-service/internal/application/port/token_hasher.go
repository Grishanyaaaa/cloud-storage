package port

type TokenHasher interface {
	Hash(token string) string
}
