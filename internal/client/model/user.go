package model

import (
	"github.com/google/uuid"
)

type Account struct {
	ID         *uuid.UUID
	Username   string
	Credential string `json:"-"`
}
