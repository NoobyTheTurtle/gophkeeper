package model

import (
	"time"

	"github.com/google/uuid"
)

type DataRecord struct {
	ID         int        `json:"id"`
	AccountID  uuid.UUID  `json:"account_id"`
	CategoryID int        `json:"category_id"`
	Name       string     `json:"name"`
	Payload    []byte     `json:"payload"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"deleted_at"`
}
