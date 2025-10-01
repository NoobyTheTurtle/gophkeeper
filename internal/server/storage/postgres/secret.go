package postgres

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

type DataVaultStore struct {
	conn *pgx.Conn
}

const (
	CreateData = `
					insert into secrets (user_id, type_id, title, content, created_at, updated_at)
					values ($1,$2,$3,$4,$5,$6)
					returning id
`
	GetData = `select id, user_id, type_id, title, content, created_at, updated_at, deleted_at
				 from secrets
				 where id = $1 and user_id = $2
`
	DeleteData = `delete from secrets where id = $1 and user_id = $2 returning id`
	UpdateData = `update secrets
					set title = $1, content = $2, updated_at = $3
					where id = $4 and user_id = $5
					returning type_id, updated_at, deleted_at
`
	DataByCategory = `select id, user_id, type_id, title, content, created_at, updated_at, deleted_at
					 from secrets
					 where type_id = $1 and user_id = $2
`
)

func NewDataVaultStore(c *pgx.Conn) *DataVaultStore {
	return &DataVaultStore{conn: c}
}

func (s *DataVaultStore) StoreData(ctx context.Context, data model.DataRecord) (model.DataRecord, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := s.conn.QueryRow(ctxWithTimeOut, CreateData, data.AccountID, data.CategoryID, data.Name,
		hex.EncodeToString(data.Payload), data.CreatedAt, data.UpdatedAt,
	).Scan(&data.ID)
	if err != nil {
		return data, fmt.Errorf("error in storing data in db: %w", err)
	}

	return data, nil
}

func (s *DataVaultStore) FetchData(ctx context.Context, data model.DataRecord) (model.DataRecord, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err := s.conn.QueryRow(ctxWithTimeOut, GetData, data.ID, data.AccountID).Scan(
		&data.ID, &data.AccountID, &data.CategoryID, &data.Name, &data.Payload,
		&data.CreatedAt, &data.UpdatedAt, &data.DeletedAt,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return data, fmt.Errorf("data getting error: %w", err)
		}

		return data, fmt.Errorf("error in getting data from db: %w", err)
	}

	decode, decErr := hex.DecodeString(string(data.Payload))
	if decErr != nil {
		return data, fmt.Errorf("error in decoding payload from db: %w", decErr)
	}

	data.Payload = decode

	return data, nil
}

func (s *DataVaultStore) RemoveData(ctx context.Context, data model.DataRecord) (model.DataRecord, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var deletedDataId *int
	err := s.conn.QueryRow(ctxWithTimeOut, DeleteData, data.ID, data.AccountID).Scan(&deletedDataId)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return data, fmt.Errorf("data deletion err: %w", err)
		}

		return data, err
	}

	return model.DataRecord{}, nil
}

func (s *DataVaultStore) UpdateData(ctx context.Context, data model.DataRecord, isForce bool) (model.DataRecord, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	existingData, _ := s.FetchData(ctx, data)

	if existingData.UpdatedAt.Unix() != data.UpdatedAt.Unix() && !isForce {
		return data, errorx.ErrUpdatedAtDoesntMatch
	}

	err := s.conn.QueryRow(ctxWithTimeOut, UpdateData, data.Name, hex.EncodeToString(data.Payload),
		time.Now(), data.ID, data.AccountID,
	).Scan(&data.CategoryID, &data.UpdatedAt, &data.DeletedAt)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return data, fmt.Errorf("data updating error: %w", err)
		}

		return data, err
	}

	return data, nil
}

func (s *DataVaultStore) QueryDataByCategory(
	ctx context.Context, category model.DataCategory, account model.Account,
) ([]model.DataRecord, error) {
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	var dataRecords []model.DataRecord

	rows, err := s.conn.Query(ctxWithTimeOut, DataByCategory, category.ID, account.ID)
	if err != nil {
		return dataRecords, fmt.Errorf("getting list of data records error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var data model.DataRecord

		if scanErr := rows.Scan(
			&data.ID,
			&data.AccountID,
			&data.CategoryID,
			&data.Name,
			&data.Payload,
			&data.CreatedAt,
			&data.UpdatedAt,
			&data.DeletedAt,
		); scanErr != nil {
			return dataRecords, fmt.Errorf("error in scanning gotten row: %w", scanErr)
		}

		data.Payload, err = hex.DecodeString(string(data.Payload))
		if err != nil {
			return dataRecords, fmt.Errorf("error in decodeing payload from row: %w", err)
		}

		dataRecords = append(dataRecords, data)
	}

	return dataRecords, nil
}
