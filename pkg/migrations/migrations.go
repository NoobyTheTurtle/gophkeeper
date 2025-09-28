package migrations

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"

	"github.com/smanhack/gophkeeper/internal/server/config"
)

type migrationManager struct {
	cfg config.Config
}

func NewMigrationManager(c config.Config) *migrationManager {
	return &migrationManager{cfg: c}
}

func (m *migrationManager) Up() error {
	migration, err := migrate.New("file://internal/server/migrations", m.cfg.DSN)
	if err != nil {
		return fmt.Errorf("mingrate.New error: %w", err)
	}
	defer func() {
		if _, closeErr := migration.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close migration: %v\n", closeErr)
		}
	}()

	if err := migration.Up(); err != nil {
		return fmt.Errorf("migration.Up error: %w", err)
	}

	return nil
}

func (m *migrationManager) RefreshTest() error {
	if err := m.DropTest(); err != nil {
		return err
	}
	if err := m.UpTest(); err != nil {
		return err
	}
	return nil
}

func (m *migrationManager) DropTest() error {
	migration, err := migrate.New("file://internal/server/migrations", m.cfg.DSNTest)
	if err != nil {
		return fmt.Errorf("mingrate.New error: %w", err)
	}
	defer func() {
		if _, closeErr := migration.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close migration: %v\n", closeErr)
		}
	}()

	if err = migration.Drop(); err != nil {
		return fmt.Errorf("error in migrating db: %w", err)
	}

	return nil
}

func (m *migrationManager) UpTest() error {
	migration, err := migrate.New("file://internal/server/migrations", m.cfg.DSNTest)
	if err != nil {
		return fmt.Errorf("mingrate.New error: %w", err)
	}
	defer func() {
		if _, closeErr := migration.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close migration: %v\n", closeErr)
		}
	}()

	if err = migration.Up(); err != nil {
		return fmt.Errorf("error in migrating db: %w", err)
	}

	return nil
}
