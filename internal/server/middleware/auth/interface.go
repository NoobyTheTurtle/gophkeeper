package auth

import (
	"github.com/smanhack/gophkeeper/pkg/crypt"
	"github.com/smanhack/gophkeeper/pkg/jwt"
)

type Crypter interface {
	Decode(sha string) (string, error)
}

type JWTManager interface {
	Decode(token string) (string, error)
}

var (
	_ Crypter    = (*crypt.Сrypt)(nil)
	_ JWTManager = (*jwt.JWT)(nil)
)
