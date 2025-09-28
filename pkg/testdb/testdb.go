package testdb

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
)

const (
	DefaultTestDSN = "postgres://gophkeeper:password@localhost:5432/gophkeeper_test?sslmode=disable"
	MigrationsPath = "internal/server/migrations"
)

type TestDB struct {
	conn *pgx.Conn
	dsn  string
}

func New(dsn string) (*TestDB, error) {
	if dsn == "" {
		dsn = getTestDSN()
	}

	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to test database: %w", err)
	}

	return &TestDB{
		conn: conn,
		dsn:  dsn,
	}, nil
}

func getTestDSN() string {
	if dsn := os.Getenv("TEST_DSN"); dsn != "" {
		return dsn
	}
	return DefaultTestDSN
}

func Setup(t *testing.T) *TestDB {
	t.Helper()

	if err := createTestDatabase(); err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	testDB, err := New("")
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	if err := testDB.RunMigrations(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	t.Cleanup(func() {
		if closeErr := testDB.Close(); closeErr != nil {
			t.Logf("Warning: failed to close test database: %v", closeErr)
		}
	})

	return testDB
}

func createTestDatabase() error {
	adminDSN := "postgres://gophkeeper:password@localhost:5432/postgres?sslmode=disable"
	conn, err := pgx.Connect(context.Background(), adminDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	defer func() {
		if closeErr := conn.Close(context.Background()); closeErr != nil {
			log.Printf("Warning: failed to close admin connection: %v", closeErr)
		}
	}()

	var exists bool
	err = conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)",
		"gophkeeper_test").Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		_, err = conn.Exec(context.Background(), "CREATE DATABASE gophkeeper_test")
		if err != nil {
			return fmt.Errorf("failed to create test database: %w", err)
		}
		log.Println("Test database 'gophkeeper_test' created")
	}

	return nil
}

func (tdb *TestDB) RunMigrations() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	projectRoot := cwd
	for {
		if _, err := os.Stat(filepath.Join(projectRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			return fmt.Errorf("could not find project root (go.mod)")
		}
		projectRoot = parent
	}

	migrationsPath := filepath.Join(projectRoot, MigrationsPath)

	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return fmt.Errorf("migrations directory not found: %s", migrationsPath)
	}

	db := stdlib.OpenDB(*tdb.conn.Config())
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("Warning: failed to close database connection: %v", closeErr)
		}
	}()

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer func() {
		if _, closeErr := m.Close(); closeErr != nil {
			log.Printf("Warning: failed to close migrate instance: %v", closeErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (tdb *TestDB) TruncateAllTables() error {
	ctx := context.Background()

	rows, err := tdb.conn.Query(ctx, `
		SELECT tablename
		FROM pg_tables
		WHERE schemaname = 'public'
		AND tablename NOT LIKE 'schema_migrations'
	`)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return fmt.Errorf("failed to scan table name: %w", err)
		}
		tables = append(tables, tableName)
	}

	if len(tables) > 0 {
		for _, table := range tables {
			_, err = tdb.conn.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
			if err != nil {
				return fmt.Errorf("failed to truncate table %s: %w", table, err)
			}
		}
	}

	_, err = tdb.conn.Exec(ctx, `
		INSERT INTO secret_types (title)
		VALUES ('login/pass'), ('text'), ('binary'), ('card')
		ON CONFLICT DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to insert default secret types: %w", err)
	}

	return nil
}

func (tdb *TestDB) Conn() *pgx.Conn {
	return tdb.conn
}

func (tdb *TestDB) Close() error {
	if tdb.conn != nil {
		return tdb.conn.Close(context.Background())
	}
	return nil
}

func (tdb *TestDB) CreateUser(login, password string) (string, error) {
	ctx := context.Background()

	var userID string
	err := tdb.conn.QueryRow(ctx,
		"INSERT INTO users (login, password) VALUES ($1, crypt($2, gen_salt('bf'))) RETURNING id",
		login, password).Scan(&userID)
	if err != nil {
		return "", fmt.Errorf("failed to create test user: %w", err)
	}

	return userID, nil
}

func (tdb *TestDB) CreateSecret(userID string, typeID int64, title string, content []byte) (int64, error) {
	ctx := context.Background()

	hexContent := hex.EncodeToString(content)

	var secretID int64
	err := tdb.conn.QueryRow(ctx,
		"INSERT INTO secrets (user_id, type_id, title, content) VALUES ($1, $2, $3, $4) RETURNING id",
		userID, typeID, title, hexContent).Scan(&secretID)
	if err != nil {
		return 0, fmt.Errorf("failed to create test secret: %w", err)
	}

	return secretID, nil
}

func WaitForDB(dsn string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for database to be ready")
		case <-ticker.C:
			conn, err := pgx.Connect(context.Background(), dsn)
			if err != nil {
				continue
			}

			if err := conn.Ping(context.Background()); err != nil {
				if closeErr := conn.Close(context.Background()); closeErr != nil {
					log.Printf("Warning: failed to close connection: %v", closeErr)
				}
				continue
			}

			if closeErr := conn.Close(context.Background()); closeErr != nil {
				log.Printf("Warning: failed to close connection: %v", closeErr)
			}
			return nil
		}
	}
}
