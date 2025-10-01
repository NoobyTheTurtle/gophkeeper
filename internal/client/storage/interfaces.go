package storage

import (
	"context"

	"github.com/smanhack/gophkeeper/internal/client/model"
	"github.com/smanhack/gophkeeper/pkg/crypt"
)

type Memorier interface {
	SetLoginPassSecrets([]model.LoginPassSecret)
	SetCardSecrets([]model.CardSecret)
	SetTextSecrets([]model.TextSecret)
	ResetStorage()
}

type Syncer interface {
	SyncAll(ctx context.Context)
}

type Crypter interface {
	Decode(sha string) (string, error)
}

var (
	_ Crypter  = (*crypt.Сrypt)(nil)
	_ Memorier = (*MemoryStorage)(nil)
)
