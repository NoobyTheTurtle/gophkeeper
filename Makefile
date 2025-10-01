# GophKeeper Makefile
# Автоматизация сборки, тестирования и развертывания

# Переменные
BINARY_NAME_SERVER=gophkeeper-server
BINARY_NAME_CLIENT=gophkeeper-client
VERSION?=dev
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.BuildVersion=$(VERSION) -X main.BuildDate=$(BUILD_DATE)"

# Директории
BUILD_DIR=build
PROTO_DIR=api/proto
PROTO_OUTPUT_DIR=pkg/api
MIGRATIONS_DIR=internal/server/migrations

# База данных
DB_DSN=postgres://gophkeeper:password@localhost:5432/gophkeeper?sslmode=disable
DB_TEST_DSN=postgres://gophkeeper:password@localhost:5432/gophkeeper_test?sslmode=disable
POSTGRES_CONTAINER=gophkeeper-postgres

.PHONY: help setup clean build build-server build-client build-client-all run-server run-client \
        proto test test-coverage test-db test-integration fmt lint tidy deps install-tools \
        postgres-up postgres-down postgres-logs postgres-restart \
        migrate-up migrate-down migrate-status migrate-reset \
        test-db-create test-db-drop test-db-reset test-migrate-up test-migrate-down

# Помощь
help: ## Показать справку по командам
	@echo ""
	@echo "GophKeeper - Makefile команды:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'
	@echo ""

# =============================================================================
# НАСТРОЙКА ПРОЕКТА
# =============================================================================

setup: postgres-up migrate-up ## Полная настройка проекта (postgres + migrations)
	@echo "✅ Проект GophKeeper настроен и готов к работе"

deps: ## Установить зависимости
	@echo "Установка зависимостей..."
	go mod download
	go mod tidy

install-tools: ## Установить инструменты разработки
	@echo "Установка инструментов разработки..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# =============================================================================
# POSTGRESQL УПРАВЛЕНИЕ
# =============================================================================

postgres-up: ## Запустить PostgreSQL контейнер
	@echo "Запуск PostgreSQL контейнера..."
	@if [ "$$(docker ps -aq -f name=$(POSTGRES_CONTAINER))" ]; then \
		if [ "$$(docker ps -aq -f name=$(POSTGRES_CONTAINER) -f status=exited)" ]; then \
			echo "Запуск остановленного контейнера..."; \
			docker start $(POSTGRES_CONTAINER); \
		else \
			echo "PostgreSQL контейнер уже запущен"; \
		fi \
	else \
		echo "Создание нового PostgreSQL контейнера..."; \
		docker run --name $(POSTGRES_CONTAINER) \
			-e POSTGRES_DB=gophkeeper \
			-e POSTGRES_USER=gophkeeper \
			-e POSTGRES_PASSWORD=password \
			-p 5432:5432 \
			-d postgres:15-alpine; \
		echo "Ожидание готовности PostgreSQL..."; \
		sleep 5; \
	fi

postgres-down: ## Остановить PostgreSQL контейнер
	@echo "Остановка PostgreSQL контейнера..."
	@docker stop $(POSTGRES_CONTAINER) 2>/dev/null || echo "Контейнер уже остановлен"

postgres-logs: ## Просмотр логов PostgreSQL
	@echo "Логи PostgreSQL контейнера:"
	@docker logs -f $(POSTGRES_CONTAINER)

postgres-restart: postgres-down postgres-up ## Перезапустить PostgreSQL контейнер

# =============================================================================
# МИГРАЦИИ БД
# =============================================================================

migrate-up: ## Применить все миграции
	@echo "Применение миграций..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	@sleep 2
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

migrate-down: ## Откатить последнюю миграцию
	@echo "Откат последней миграции..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down 1

migrate-status: ## Показать статус миграций
	@echo "Статус миграций:"
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" version

migrate-reset: ## Сбросить все миграции и применить заново
	@echo "Сброс всех миграций..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	@echo "Откат всех миграций..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" down -all || echo "Миграции уже отсутствуют"
	@echo "Применение всех миграций заново..."
	@migrate -path $(MIGRATIONS_DIR) -database "$(DB_DSN)" up

# =============================================================================
# PROTOBUF
# =============================================================================

proto: ## Генерировать protobuf код
	@echo "Генерация protobuf кода..."
	@if [ ! -d "$(PROTO_DIR)" ]; then \
		echo "❌ Директория $(PROTO_DIR) не найдена"; \
		exit 1; \
	fi
	@echo "Создание директории $(PROTO_OUTPUT_DIR) если не существует..."
	@mkdir -p $(PROTO_OUTPUT_DIR)
	@echo "Удаление старых сгенерированных файлов..."
	@find $(PROTO_OUTPUT_DIR) -name "*.pb.go" -delete 2>/dev/null || true
	@echo "Генерация Go кода из protobuf схем..."
	protoc --proto_path=$(PROTO_DIR) \
		--proto_path=/usr/local/include \
		--go_out=$(PROTO_OUTPUT_DIR) \
		--go-grpc_out=$(PROTO_OUTPUT_DIR) \
		--go_opt=paths=source_relative \
		--go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/user.proto \
		$(PROTO_DIR)/secret.proto \
		$(PROTO_DIR)/secret_type.proto
	@echo "✅ Protobuf код сгенерирован успешно"

# =============================================================================
# СБОРКА
# =============================================================================

build: build-server build-client ## Собрать сервер и клиент

build-server: ## Собрать сервер
	@echo "Сборка сервера..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_SERVER) ./cmd/server

build-client: ## Собрать клиент для текущей платформы
	@echo "Сборка клиента..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME_CLIENT) ./cmd/client

build-client-all: ## Собрать клиент для всех платформ
	@echo "Кроссплатформенная сборка клиента..."
	@mkdir -p $(BUILD_DIR)/windows
	@mkdir -p $(BUILD_DIR)/linux
	@mkdir -p $(BUILD_DIR)/darwin
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/windows/$(BINARY_NAME_CLIENT).exe ./cmd/client
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/linux/$(BINARY_NAME_CLIENT) ./cmd/client
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/$(BINARY_NAME_CLIENT) ./cmd/client
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/darwin/$(BINARY_NAME_CLIENT)-arm64 ./cmd/client

# =============================================================================
# ЗАПУСК
# =============================================================================

run-server: ## Запустить сервер локально
	@echo "Запуск сервера..."
	go run $(LDFLAGS) ./cmd/server

run-client: ## Запустить клиент
	@echo "Запуск клиента..."
	go run $(LDFLAGS) ./cmd/client

# =============================================================================
# ТЕСТЕВАЯ БАЗА ДАННЫХ
# =============================================================================

test-db-create: ## Создать тестовую базу данных
	@echo "Создание тестовой базы данных..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	@sleep 1
	@docker exec -i $(POSTGRES_CONTAINER) psql -U gophkeeper -d postgres -c "CREATE DATABASE gophkeeper_test;" 2>/dev/null || echo "База данных gophkeeper_test уже существует"
	@echo "✅ Тестовая база данных готова"

test-db-drop: ## Удалить тестовую базу данных
	@echo "Удаление тестовой базы данных..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	@docker exec -i $(POSTGRES_CONTAINER) psql -U gophkeeper -d postgres -c "DROP DATABASE IF EXISTS gophkeeper_test;"
	@echo "✅ Тестовая база данных удалена"

test-db-reset: test-db-drop test-db-create test-migrate-up ## Пересоздать тестовую базу данных
	@echo "✅ Тестовая база данных пересоздана"

test-migrate-up: test-db-create ## Применить миграции к тестовой БД
	@echo "Применение миграций к тестовой базе данных..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	@sleep 2
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_TEST_DSN)" up
	@echo "✅ Миграции применены к тестовой БД"

test-migrate-down: ## Откатить миграции тестовой БД
	@echo "Откат миграций тестовой базы данных..."
	@if ! docker ps | grep -q $(POSTGRES_CONTAINER); then \
		echo "❌ PostgreSQL не запущен. Запустите 'make postgres-up'"; \
		exit 1; \
	fi
	migrate -path $(MIGRATIONS_DIR) -database "$(DB_TEST_DSN)" down -all

# =============================================================================
# ТЕСТИРОВАНИЕ
# =============================================================================

test: ## Запустить unit тесты
	@echo "Запуск unit тестов..."
	go test -v -short ./...

test-db: test-db-reset ## Запустить интеграционные тесты с БД
	@echo "Запуск интеграционных тестов с тестовой базой данных..."
	@export TEST_DSN=$(DB_TEST_DSN) && go test -v ./internal/server/storage/postgres/...

test-integration: test-db-reset ## Запустить все интеграционные тесты
	@echo "Запуск всех интеграционных тестов..."
	@export TEST_DSN=$(DB_TEST_DSN) && go test -v -tags=integration ./...

test-coverage: test-db-reset ## Запустить тесты с покрытием
	@echo "Запуск тестов с покрытием..."
	@export TEST_DSN=$(DB_TEST_DSN) && go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Отчет покрытия сохранен в coverage.html"

test-all: test test-db ## Запустить все тесты (unit + integration)

# =============================================================================
# ЛИНТЕРЫ И ФОРМАТИРОВАНИЕ
# =============================================================================

fmt: ## Форматировать код
	@echo "Форматирование кода..."
	go fmt ./...

lint: ## Запустить линтер
	@echo "Запуск линтера..."
	@if ! command -v golangci-lint > /dev/null; then \
		echo "golangci-lint не установлен. Установка..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

tidy: ## Очистить зависимости
	@echo "Очистка зависимостей..."
	go mod tidy

# =============================================================================
# ОЧИСТКА
# =============================================================================

clean: ## Очистить артефакты сборки и контейнеры
	@echo "Очистка артефактов сборки..."
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "Остановка и удаление PostgreSQL контейнера..."
	@docker stop $(POSTGRES_CONTAINER) 2>/dev/null || true
	@docker rm $(POSTGRES_CONTAINER) 2>/dev/null || true
	@echo "Очистка завершена"

# =============================================================================
# ПОЛНАЯ ПЕРЕСБОРКА
# =============================================================================

rebuild: clean deps proto build ## Полная очистка и пересборка