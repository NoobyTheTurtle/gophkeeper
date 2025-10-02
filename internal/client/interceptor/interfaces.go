package interceptor

type TokenProvider interface {
	GetToken() string
}
