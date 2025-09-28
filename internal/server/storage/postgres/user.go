package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"

	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/internal/server/storage"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

var _ storage.AccountServerStorage = (*AccountStore)(nil)

type AccountStore struct {
	conn *pgx.Conn
}

const (
	CreateAccount     = `INSERT INTO users (login, password) VALUES ($1, crypt($2, gen_salt('bf'))) returning id`
	GetAccountId      = `SELECT id FROM users WHERE login = $1 AND password = crypt($2, password)`
	DeleteAccountById = `DELETE from users where id = $1 returning login`
)

func NewAccountStore(c *pgx.Conn) *AccountStore {
	return &AccountStore{conn: c}
}

func (u AccountStore) StoreAccount(ctx context.Context, account model.Account) (model.Account, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := u.conn.QueryRow(ctxWithTimeOut, CreateAccount, account.Username, account.Credential).Scan(&account.ID)
	if err != nil {
		if pgErr, ok := err.(*pgconn.PgError); ok {
			if pgerrcode.IsIntegrityConstraintViolation(pgErr.Code) {
				return account, errorx.ErrConflict
			}
		}

		return account, fmt.Errorf("account insertion err: %w", err)
	}

	return account, nil
}

func (u AccountStore) FindByCredentials(ctx context.Context, account model.Account) (model.Account, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := u.conn.QueryRow(ctxWithTimeOut, GetAccountId, account.Username, account.Credential).Scan(&account.ID)
	if err != nil {
		return account, fmt.Errorf("account login err: %w", err)
	}

	return account, nil
}

func (u AccountStore) RemoveAccount(ctx context.Context, account model.Account) (model.Account, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var deletedLogin *string
	err := u.conn.QueryRow(ctxWithTimeOut, DeleteAccountById, account.ID).Scan(&deletedLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return account, err
		}

		return account, fmt.Errorf("account deletion err: %w", err)
	}

	return model.Account{}, nil
}
