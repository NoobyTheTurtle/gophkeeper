package storage

import (
	"context"

	"github.com/smanhack/gophkeeper/internal/server/model"
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
