package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCategoryStore(t *testing.T) {
	helper := NewTestHelper(t)

	store := NewCategoryStore(helper.GetConn())

	assert.NotNil(t, store)
	assert.NotNil(t, store.conn)
}

func TestCategoryStore_ListCategories(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewCategoryStore(helper.GetConn())

	t.Run("should return default secret types", func(t *testing.T) {
		ctx := context.Background()

		categories, err := store.ListCategories(ctx)

		require.NoError(t, err)
		assert.Len(t, categories, 4, "Should have 4 default secret types")

		// Проверяем что есть все основные типы
		typeNames := make(map[string]bool)
		for _, category := range categories {
			typeNames[category.Name] = true
			assert.Greater(t, category.ID, uint(0), "ID should be greater than 0")
		}

		assert.True(t, typeNames["login/pass"], "Should contain login/pass type")
		assert.True(t, typeNames["text"], "Should contain text type")
		assert.True(t, typeNames["binary"], "Should contain binary type")
		assert.True(t, typeNames["card"], "Should contain card type")
	})

	t.Run("should return empty list when no types exist", func(t *testing.T) {
		// Очищаем таблицу secret_types для этого теста
		helper.ExecuteSQL(t, "DELETE FROM secret_types")

		ctx := context.Background()
		categories, err := store.ListCategories(ctx)

		require.NoError(t, err)
		assert.Empty(t, categories, "Should return empty list when no types exist")

		// Восстанавливаем данные для других тестов
		helper.ExecuteSQL(t,
			"INSERT INTO secret_types (title) VALUES ('login/pass'), ('text'), ('binary'), ('card')")
	})
}

func TestCategoryStore_ListCategories_Integration(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewCategoryStore(helper.GetConn())
	ctx := context.Background()

	// Добавляем кастомный тип секрета
	helper.ExecuteSQL(t, "INSERT INTO secret_types (title) VALUES ('custom_type')")

	categories, err := store.ListCategories(ctx)

	require.NoError(t, err)
	assert.Len(t, categories, 5, "Should have 4 default + 1 custom type")

	// Проверяем что кастомный тип присутствует
	found := false
	for _, category := range categories {
		if category.Name == "custom_type" {
			found = true
			break
		}
	}
	assert.True(t, found, "Should contain custom type")
}

func TestCategoryStore_ListCategories_Performance(t *testing.T) {
	helper := NewTestHelper(t)
	helper.CleanupTables(t)

	store := NewCategoryStore(helper.GetConn())
	ctx := context.Background()

	// Добавляем много типов для проверки производительности
	for i := 0; i < 100; i++ {
		helper.ExecuteSQL(t, "INSERT INTO secret_types (title) VALUES ($1)",
			"type_"+string(rune('A'+i%26))+string(rune('0'+i%10)))
	}

	categories, err := store.ListCategories(ctx)

	require.NoError(t, err)
	assert.Len(t, categories, 104, "Should have 4 default + 100 additional types")

	// Проверяем что все записи корректно отсканированы
	for _, category := range categories {
		assert.Greater(t, category.ID, uint(0))
		assert.NotEmpty(t, category.Name)
	}
}
