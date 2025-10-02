package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

func TestNewDataVaultStore(t *testing.T) {
	helper := NewTestHelper(t)

	store := NewDataVaultStore(helper.GetConn())

	assert.NotNil(t, store)
	assert.NotNil(t, store.conn)
}

func TestDataVaultStore_StoreData(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())

	// Создаем тестового пользователя
	userID := helper.CreateTestUser(t, "secretuser", "secretpass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("should store secret successfully", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)
		data := model.DataRecord{
			AccountID:  uid,
			CategoryID: 1, // login/pass type
			Name:       "test secret",
			Payload:    []byte("encrypted test content"),
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		ctx := context.Background()
		result, err := store.StoreData(ctx, data)

		require.NoError(t, err)
		assert.Greater(t, result.ID, 0)
		assert.Equal(t, uid, result.AccountID)
		assert.Equal(t, 1, result.CategoryID)
		assert.Equal(t, "test secret", result.Name)
		assert.Equal(t, []byte("encrypted test content"), result.Payload)
		assert.Equal(t, now, result.CreatedAt)
		assert.Equal(t, now, result.UpdatedAt)
	})

	t.Run("should handle binary payload", func(t *testing.T) {
		binaryData := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		data := model.DataRecord{
			AccountID:  uid,
			CategoryID: 3, // binary type
			Name:       "binary secret",
			Payload:    binaryData,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		ctx := context.Background()
		result, err := store.StoreData(ctx, data)

		require.NoError(t, err)
		assert.Greater(t, result.ID, 0)
		assert.Equal(t, binaryData, result.Payload)
	})
}

func TestDataVaultStore_FetchData(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())

	// Создаем тестового пользователя
	userID := helper.CreateTestUser(t, "fetchuser", "fetchpass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("should fetch secret successfully", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 1, "fetch secret", []byte("fetch content"))

		data := model.DataRecord{
			ID:        int(secretID),
			AccountID: uid,
		}

		ctx := context.Background()
		result, err := store.FetchData(ctx, data)

		require.NoError(t, err)
		assert.Equal(t, int(secretID), result.ID)
		assert.Equal(t, uid, result.AccountID)
		assert.Equal(t, 1, result.CategoryID)
		assert.Equal(t, "fetch secret", result.Name)
		// Контент будет в hex-кодировке в БД, но декодирован при получении
		assert.NotEmpty(t, result.Payload)
	})

	t.Run("should return error for non-existent secret", func(t *testing.T) {
		data := model.DataRecord{
			ID:        99999, // несуществующий ID
			AccountID: uid,
		}

		ctx := context.Background()
		_, err := store.FetchData(ctx, data)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error in getting data from db")
	})

	t.Run("should return error for wrong user", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 1, "wrong user secret", []byte("content"))

		// Пытаемся получить с другим пользователем
		wrongUID := uuid.New()
		data := model.DataRecord{
			ID:        int(secretID),
			AccountID: wrongUID,
		}

		ctx := context.Background()
		_, err := store.FetchData(ctx, data)

		assert.Error(t, err)
	})
}

func TestDataVaultStore_RemoveData(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())

	// Создаем тестового пользователя
	userID := helper.CreateTestUser(t, "removeuser", "removepass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("should remove secret successfully", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 1, "remove secret", []byte("remove content"))

		data := model.DataRecord{
			ID:        int(secretID),
			AccountID: uid,
		}

		ctx := context.Background()
		result, err := store.RemoveData(ctx, data)

		require.NoError(t, err)
		assert.Zero(t, result.ID)
		assert.True(t, result.AccountID.String() == "00000000-0000-0000-0000-000000000000")

		// Проверяем что секрет удален
		var exists bool
		err = helper.GetConn().QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM secrets WHERE id = $1)", secretID).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "Secret should be deleted")
	})

	t.Run("should return error for non-existent secret", func(t *testing.T) {
		data := model.DataRecord{
			ID:        99999,
			AccountID: uid,
		}

		ctx := context.Background()
		_, err := store.RemoveData(ctx, data)

		assert.Error(t, err)
	})
}

func TestDataVaultStore_UpdateData(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())

	// Создаем тестового пользователя
	userID := helper.CreateTestUser(t, "updateuser", "updatepass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("should update secret successfully", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 1, "original secret", []byte("original content"))

		// Получаем данные секрета для правильного updated_at
		fetchData := model.DataRecord{ID: int(secretID), AccountID: uid}
		existing, err := store.FetchData(context.Background(), fetchData)
		require.NoError(t, err)

		// Обновляем секрет
		updateData := model.DataRecord{
			ID:        int(secretID),
			AccountID: uid,
			Name:      "updated secret",
			Payload:   []byte("updated content"),
			UpdatedAt: existing.UpdatedAt, // используем правильный timestamp
		}

		ctx := context.Background()
		result, err := store.UpdateData(ctx, updateData, false)

		require.NoError(t, err)
		assert.Equal(t, int(secretID), result.ID)
		assert.Equal(t, uid, result.AccountID)
	})

	t.Run("should return conflict error for outdated timestamp", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 2, "conflict secret", []byte("conflict content"))

		// Пытаемся обновить с неправильным timestamp
		updateData := model.DataRecord{
			ID:        int(secretID),
			AccountID: uid,
			Name:      "updated secret",
			Payload:   []byte("updated content"),
			UpdatedAt: time.Now().Add(-time.Hour), // старый timestamp
		}

		ctx := context.Background()
		_, err := store.UpdateData(ctx, updateData, false)

		assert.ErrorIs(t, err, errorx.ErrUpdatedAtDoesntMatch)
	})

	t.Run("should force update with wrong timestamp", func(t *testing.T) {
		// Создаем секрет
		secretID := helper.CreateTestSecret(t, userID, 2, "force secret", []byte("force content"))

		// Принудительно обновляем с неправильным timestamp
		updateData := model.DataRecord{
			ID:        int(secretID),
			AccountID: uid,
			Name:      "force updated secret",
			Payload:   []byte("force updated content"),
			UpdatedAt: time.Now().Add(-time.Hour), // старый timestamp
		}

		ctx := context.Background()
		result, err := store.UpdateData(ctx, updateData, true) // force = true

		require.NoError(t, err)
		assert.Equal(t, int(secretID), result.ID)
	})
}

func TestDataVaultStore_QueryDataByCategory(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())

	// Создаем тестового пользователя
	userID := helper.CreateTestUser(t, "queryuser", "querypass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("should return secrets by category", func(t *testing.T) {
		// Создаем несколько секретов разных типов
		helper.CreateTestSecret(t, userID, 1, "login secret 1", []byte("content1"))
		helper.CreateTestSecret(t, userID, 1, "login secret 2", []byte("content2"))
		helper.CreateTestSecret(t, userID, 2, "text secret", []byte("text content"))

		category := model.DataCategory{ID: 1} // login/pass
		account := model.Account{ID: &uid}

		ctx := context.Background()
		results, err := store.QueryDataByCategory(ctx, category, account)

		require.NoError(t, err)
		assert.Len(t, results, 2, "Should return 2 login/pass secrets")

		for _, result := range results {
			assert.Equal(t, uid, result.AccountID)
			assert.Equal(t, 1, result.CategoryID)
			assert.NotEmpty(t, result.Name)
			assert.NotEmpty(t, result.Payload)
		}
	})

	t.Run("should return empty list for non-existent category", func(t *testing.T) {
		category := model.DataCategory{ID: 999} // несуществующий тип
		account := model.Account{ID: &uid}

		ctx := context.Background()
		results, err := store.QueryDataByCategory(ctx, category, account)

		require.NoError(t, err)
		assert.Empty(t, results)
	})

	t.Run("should return only user's secrets", func(t *testing.T) {
		// Создаем другого пользователя и его секрет
		otherUserID := helper.CreateTestUser(t, "otheruser", "otherpass")
		helper.CreateTestSecret(t, otherUserID, 1, "other user secret", []byte("other content"))

		category := model.DataCategory{ID: 1}
		account := model.Account{ID: &uid}

		ctx := context.Background()
		results, err := store.QueryDataByCategory(ctx, category, account)

		require.NoError(t, err)
		// Должны получить только секреты текущего пользователя
		for _, result := range results {
			assert.Equal(t, uid, result.AccountID, "Should only return current user's secrets")
		}
	})
}

func TestDataVaultStore_Integration(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewDataVaultStore(helper.GetConn())
	ctx := context.Background()

	// Создаем пользователя
	userID := helper.CreateTestUser(t, "integrationuser", "integrationpass")
	uid, err := uuid.Parse(userID)
	require.NoError(t, err)

	t.Run("full secret lifecycle", func(t *testing.T) {
		now := time.Now().Truncate(time.Second)

		// 1. Создаем секрет
		data := model.DataRecord{
			AccountID:  uid,
			CategoryID: 1,
			Name:       "integration secret",
			Payload:    []byte("integration content"),
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		created, err := store.StoreData(ctx, data)
		require.NoError(t, err)
		require.Greater(t, created.ID, 0)

		// 2. Получаем секрет
		fetchData := model.DataRecord{ID: created.ID, AccountID: uid}
		fetched, err := store.FetchData(ctx, fetchData)
		require.NoError(t, err)
		assert.Equal(t, created.ID, fetched.ID)
		assert.Equal(t, "integration secret", fetched.Name)

		// 3. Обновляем секрет
		updateData := model.DataRecord{
			ID:        created.ID,
			AccountID: uid,
			Name:      "updated integration secret",
			Payload:   []byte("updated integration content"),
			UpdatedAt: fetched.UpdatedAt,
		}

		updated, err := store.UpdateData(ctx, updateData, false)
		require.NoError(t, err)
		assert.Equal(t, created.ID, updated.ID)

		// 4. Получаем список по категории
		category := model.DataCategory{ID: 1}
		account := model.Account{ID: &uid}
		list, err := store.QueryDataByCategory(ctx, category, account)
		require.NoError(t, err)
		assert.NotEmpty(t, list)

		// 5. Удаляем секрет
		deleteData := model.DataRecord{ID: created.ID, AccountID: uid}
		deleted, err := store.RemoveData(ctx, deleteData)
		require.NoError(t, err)
		assert.Zero(t, deleted.ID)

		// 6. Проверяем что секрет удален
		_, err = store.FetchData(ctx, fetchData)
		assert.Error(t, err)
	})
}
