package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4"

	"github.com/smanhack/gophkeeper/internal/server/model"
)

type CategoryStore struct {
	conn *pgx.Conn
}

const (
	GetCategoryList = `SELECT id, title FROM secret_types`
)

func NewCategoryStore(c *pgx.Conn) *CategoryStore {
	return &CategoryStore{conn: c}
}

func (s *CategoryStore) ListCategories(ctx context.Context) ([]model.DataCategory, error) {
	var list []model.DataCategory

	rows, err := s.conn.Query(ctx, GetCategoryList)
	if err != nil {
		return list, fmt.Errorf("error in getting categories list: %w", err)
	}

	for rows.Next() {
		m := model.DataCategory{}
		if err = rows.Scan(&m.ID, &m.Name); err != nil {
			return list, fmt.Errorf("error scanning categories: %w", err)
		}

		list = append(list, m)
	}

	return list, nil
}
