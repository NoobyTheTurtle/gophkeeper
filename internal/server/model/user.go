package model

import "github.com/google/uuid"

type Account struct {
	ID         *uuid.UUID `json:"id"`
	Username   string     `json:"username" validate:"gte=3"`
	Credential string     `json:"-" validate:"gte=3"`
}
