package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v4"

	"github.com/smanhack/gophkeeper/pkg/testdb"
)

type TestHelper struct {
	testDB *testdb.TestDB
	conn   *pgx.Conn
}

func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	// Проверяем что тесты запускаются с тестовой базой
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Инициализируем тестовую базу данных
	testDB := testdb.Setup(t)

	return &TestHelper{
		testDB: testDB,
		conn:   testDB.Conn(),
	}
}

func (h *TestHelper) GetConn() *pgx.Conn {
	return h.conn
}

func (h *TestHelper) CleanupTables(t *testing.T) {
	t.Helper()

	if err := h.testDB.TruncateAllTables(); err != nil {
		t.Fatalf("Failed to cleanup tables: %v", err)
	}
}

func (h *TestHelper) CreateTestUser(t *testing.T, login, password string) string {
	t.Helper()

	userID, err := h.testDB.CreateUser(login, password)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	return userID
}

func (h *TestHelper) CreateTestSecret(t *testing.T, userID string, typeID int64, title string, content []byte) int64 {
	t.Helper()

	secretID, err := h.testDB.CreateSecret(userID, typeID, title, content)
	if err != nil {
		t.Fatalf("Failed to create test secret: %v", err)
	}

	return secretID
}

func (h *TestHelper) WaitForDB(t *testing.T) {
	t.Helper()

	err := testdb.WaitForDB(testdb.DefaultTestDSN, 30*time.Second)
	if err != nil {
		t.Fatalf("Database not ready: %v", err)
	}
}

func (h *TestHelper) AssertUserExists(t *testing.T, userID string) {
	t.Helper()

	var exists bool
	err := h.conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check user existence: %v", err)
	}

	if !exists {
		t.Fatalf("User %s does not exist", userID)
	}
}

func (h *TestHelper) AssertSecretExists(t *testing.T, secretID int64) {
	t.Helper()

	var exists bool
	err := h.conn.QueryRow(context.Background(),
		"SELECT EXISTS(SELECT 1 FROM secrets WHERE id = $1 AND deleted_at IS NULL)",
		secretID).Scan(&exists)
	if err != nil {
		t.Fatalf("Failed to check secret existence: %v", err)
	}

	if !exists {
		t.Fatalf("Secret %d does not exist", secretID)
	}
}

func (h *TestHelper) AssertSecretDeleted(t *testing.T, secretID int64) {
	t.Helper()

	var deletedAt *time.Time
	err := h.conn.QueryRow(context.Background(),
		"SELECT deleted_at FROM secrets WHERE id = $1",
		secretID).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("Failed to check secret deleted status: %v", err)
	}

	if deletedAt == nil {
		t.Fatalf("Secret %d is not marked as deleted", secretID)
	}
}

func (h *TestHelper) GetSecretTypesCount(t *testing.T) int {
	t.Helper()

	var count int
	err := h.conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM secret_types").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to get secret types count: %v", err)
	}

	return count
}

func (h *TestHelper) ExecuteSQL(t *testing.T, query string, args ...interface{}) {
	t.Helper()

	_, err := h.conn.Exec(context.Background(), query, args...)
	if err != nil {
		t.Fatalf("Failed to execute SQL: %v", err)
	}
}

func (h *TestHelper) QueryRowSQL(t *testing.T, query string, args ...interface{}) pgx.Row {
	t.Helper()

	return h.conn.QueryRow(context.Background(), query, args...)
}
