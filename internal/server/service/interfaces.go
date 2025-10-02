package service

//go:generate mockgen -package=mock -destination=mock/service_mock.go . AccountServerStorage,CategoryServerStorage,DataVaultServerStorage,Crypter,JWTManager

import (
	"context"

	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/internal/server/service/mock"
	"github.com/smanhack/gophkeeper/internal/server/storage/postgres"
	"github.com/smanhack/gophkeeper/pkg/crypt"
	"github.com/smanhack/gophkeeper/pkg/jwt"
)

type AccountServerStorage interface {
	// StoreAccount - create a new model.Account in storage.
	StoreAccount(ctx context.Context, user model.Account) (model.Account, error)
	// FindByCredentials - returns model.Account from storage.
	FindByCredentials(ctx context.Context, user model.Account) (model.Account, error)
	// RemoveAccount - deletes a user from storage.
	RemoveAccount(ctx context.Context, user model.Account) (model.Account, error)
}

type CategoryServerStorage interface {
	// ListCategories - returns list of model.DataCategory from storage.
	ListCategories(ctx context.Context) ([]model.DataCategory, error)
}

type DataVaultServerStorage interface {
	// StoreData - creates new model.DataRecord in storage.
	StoreData(ctx context.Context, secret model.DataRecord) (model.DataRecord, error)
	// FetchData - gets a model.DataRecord from storage.
	FetchData(ctx context.Context, secret model.DataRecord) (model.DataRecord, error)
	// RemoveData - deletes a model.DataRecord from storage.
	RemoveData(ctx context.Context, secret model.DataRecord) (model.DataRecord, error)
	// UpdateData - updates a model.DataRecord in storage.
	UpdateData(ctx context.Context, secret model.DataRecord, isForce bool) (model.DataRecord, error)
	// QueryDataByCategory - returns a list of []model.DataRecord from storage.
	QueryDataByCategory(ctx context.Context, secretType model.DataCategory, user model.Account) ([]model.DataRecord, error)
}

type Crypter interface {
	Encode(payload string) string
	Decode(sha string) (string, error)
}

type JWTManager interface {
	Issue(id string) (string, error)
	Decode(token string) (string, error)
}

var (
	_ AccountServerStorage = (*postgres.AccountStore)(nil)
	_ AccountServerStorage = (*mock.MockAccountServerStorage)(nil)

	_ CategoryServerStorage = (*postgres.CategoryStore)(nil)
	_ CategoryServerStorage = (*mock.MockCategoryServerStorage)(nil)

	_ DataVaultServerStorage = (*postgres.DataVaultStore)(nil)
	_ DataVaultServerStorage = (*mock.MockDataVaultServerStorage)(nil)

	_ Crypter = (*crypt.Сrypt)(nil)
	_ Crypter = (*mock.MockCrypter)(nil)

	_ JWTManager = (*jwt.JWT)(nil)
	_ JWTManager = (*mock.MockJWTManager)(nil)
)
