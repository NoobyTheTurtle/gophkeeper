package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

func TestNewAccountStore(t *testing.T) {
	helper := NewTestHelper(t)

	store := NewAccountStore(helper.GetConn())

	assert.NotNil(t, store)
	assert.NotNil(t, store.conn)
}

func TestAccountStore_StoreAccount(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewAccountStore(helper.GetConn())

	t.Run("should create account successfully", func(t *testing.T) {
		account := model.Account{
			Username:   "testuser",
			Credential: "testpassword",
		}

		ctx := context.Background()
		result, err := store.StoreAccount(ctx, account)

		require.NoError(t, err)
		assert.NotNil(t, result.ID)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "testpassword", result.Credential)

		// Проверяем что пользователь действительно создался
		helper.AssertUserExists(t, result.ID.String())
	})

	t.Run("should return conflict error for duplicate username", func(t *testing.T) {
		// Создаем первого пользователя
		account1 := model.Account{
			Username:   "duplicate_user",
			Credential: "password1",
		}

		ctx := context.Background()
		_, err := store.StoreAccount(ctx, account1)
		require.NoError(t, err)

		// Пытаемся создать второго с тем же логином
		account2 := model.Account{
			Username:   "duplicate_user", // тот же логин
			Credential: "password2",
		}

		_, err = store.StoreAccount(ctx, account2)
		assert.ErrorIs(t, err, errorx.ErrConflict)
	})

	t.Run("should handle empty credentials", func(t *testing.T) {
		account := model.Account{
			Username:   "empty_cred_user",
			Credential: "",
		}

		ctx := context.Background()
		result, err := store.StoreAccount(ctx, account)

		require.NoError(t, err)
		assert.NotNil(t, result.ID)
	})
}

func TestAccountStore_FindByCredentials(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewAccountStore(helper.GetConn())

	t.Run("should find account with correct credentials", func(t *testing.T) {
		// Создаем пользователя
		userID := helper.CreateTestUser(t, "finduser", "findpassword")

		account := model.Account{
			Username:   "finduser",
			Credential: "findpassword",
		}

		ctx := context.Background()
		result, err := store.FindByCredentials(ctx, account)

		require.NoError(t, err)
		assert.NotNil(t, result.ID)
		assert.Equal(t, userID, result.ID.String())
		assert.Equal(t, "finduser", result.Username)
	})

	t.Run("should return error for wrong password", func(t *testing.T) {
		// Создаем пользователя
		helper.CreateTestUser(t, "wrongpassuser", "correctpassword")

		account := model.Account{
			Username:   "wrongpassuser",
			Credential: "wrongpassword",
		}

		ctx := context.Background()
		_, err := store.FindByCredentials(ctx, account)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account login err")
	})

	t.Run("should return error for non-existent user", func(t *testing.T) {
		account := model.Account{
			Username:   "nonexistent",
			Credential: "anypassword",
		}

		ctx := context.Background()
		_, err := store.FindByCredentials(ctx, account)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "account login err")
	})
}

func TestAccountStore_RemoveAccount(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewAccountStore(helper.GetConn())

	t.Run("should remove account successfully", func(t *testing.T) {
		// Создаем пользователя
		userID := helper.CreateTestUser(t, "deleteuser", "deletepassword")
		uid, err := uuid.Parse(userID)
		require.NoError(t, err)

		account := model.Account{
			ID: &uid,
		}

		ctx := context.Background()
		result, err := store.RemoveAccount(ctx, account)

		require.NoError(t, err)
		assert.Nil(t, result.ID)
		assert.Empty(t, result.Username)

		// Проверяем что пользователь действительно удален
		var exists bool
		err = helper.GetConn().QueryRow(context.Background(),
			"SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userID).Scan(&exists)
		require.NoError(t, err)
		assert.False(t, exists, "User should be deleted from database")
	})

	t.Run("should return error for non-existent account", func(t *testing.T) {
		// Генерируем случайный UUID который точно не существует
		nonExistentID := uuid.New()
		account := model.Account{
			ID: &nonExistentID,
		}

		ctx := context.Background()
		_, err := store.RemoveAccount(ctx, account)

		assert.Error(t, err)
		// Проверяем что это ошибка "no rows"
		assert.Contains(t, err.Error(), "no rows")
	})

	t.Run("should handle nil ID", func(t *testing.T) {
		account := model.Account{
			ID: nil,
		}

		ctx := context.Background()
		_, err := store.RemoveAccount(ctx, account)

		assert.Error(t, err)
	})
}

func TestAccountStore_Integration(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewAccountStore(helper.GetConn())
	ctx := context.Background()

	t.Run("full user lifecycle", func(t *testing.T) {
		// 1. Создаем пользователя
		account := model.Account{
			Username:   "lifecycle_user",
			Credential: "lifecycle_password",
		}

		created, err := store.StoreAccount(ctx, account)
		require.NoError(t, err)
		require.NotNil(t, created.ID)

		// 2. Ищем его по credentials
		found, err := store.FindByCredentials(ctx, account)
		require.NoError(t, err)
		assert.Equal(t, created.ID.String(), found.ID.String())

		// 3. Удаляем пользователя
		deleted, err := store.RemoveAccount(ctx, found)
		require.NoError(t, err)
		assert.Nil(t, deleted.ID)

		// 4. Проверяем что его больше нет
		_, err = store.FindByCredentials(ctx, account)
		assert.Error(t, err)
	})
}
