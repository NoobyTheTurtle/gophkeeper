package service

import (
	"context"

	"github.com/smanhack/gophkeeper/internal/client/storage"
	proto "github.com/smanhack/gophkeeper/pkg/api"
	"github.com/smanhack/gophkeeper/pkg/crypt"
)

type Memorier interface {
	FindInStorage(id int) (interface{}, bool)
	GetDataRecordList(id int) []*proto.DataRecord
	ResetStorage()
}

type Syncer interface {
	SyncAll(ctx context.Context)
}

type Crypter interface {
	Decode(sha string) (string, error)
	Encode(payload string) string
}

var (
	_ Crypter  = (*crypt.Сrypt)(nil)
	_ Memorier = (*storage.MemoryStorage)(nil)
	_ Syncer   = (*storage.Sync)(nil)
)
