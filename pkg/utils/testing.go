// Package utils is a package that provides some helper functions.
package utils

import (
	"context"

	"github.com/jackc/pgx/v4"

	"github.com/smanhack/gophkeeper/internal/server/config"
	"github.com/smanhack/gophkeeper/pkg/migrations"
)

func CreatePostgresTestConn() *pgx.Conn {
	ctx := context.Background()
	cfg := config.NewConfig()

	con, err := pgx.Connect(ctx, cfg.DSNTest)
	if err != nil {
		panic(err)
	}

	errPing := con.Ping(ctx)
	if errPing != nil {
		panic(errPing)
	}

	return con
}

func RefreshTestDatabase() {
	migrator := migrations.NewMigrationManager(config.NewConfig())
	if err := migrator.RefreshTest(); err != nil {
		panic(err)
	}
}
