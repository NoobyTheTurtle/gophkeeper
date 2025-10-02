package app

import (
	"context"
	"fmt"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	grpczap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpcrecovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/jackc/pgx/v4"
	"go.uber.org/zap"

	"github.com/smanhack/gophkeeper/internal/server/config"
	"github.com/smanhack/gophkeeper/internal/server/middleware/auth"
	"github.com/smanhack/gophkeeper/internal/server/service"
	"github.com/smanhack/gophkeeper/internal/server/storage/postgres"
	"github.com/smanhack/gophkeeper/pkg/crypt"
	"github.com/smanhack/gophkeeper/pkg/jwt"
	"github.com/smanhack/gophkeeper/pkg/logger"
	"github.com/smanhack/gophkeeper/pkg/migrations"
	"github.com/smanhack/gophkeeper/pkg/server"
)

type App struct {
	DB     *pgx.Conn
	Server *server.GrpcServer
	Logger *zap.Logger
}

func NewApp(ctx context.Context) (*App, error) {
	cfg := config.NewConfig()
	log := logger.NewLogger()

	cr, errCr := crypt.NewCrypt()
	if errCr != nil {
		return nil, fmt.Errorf("crypt creating error: %w", errCr)
	}

	dbConn, err := pgx.Connect(ctx, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}

	if err = dbConn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	jwtManager, errJwt := jwt.NewJWT(cfg.JWTSecret, cfg.JWTExp)
	if errJwt != nil {
		return nil, fmt.Errorf("jwtManager creating error: %w", errJwt)
	}

	migrationManager := migrations.NewMigrationManager(cfg)
	err = migrationManager.Up()
	if err != nil {
		return nil, fmt.Errorf("migration error: %w", err)
	}

	usersStorage := postgres.NewAccountStore(dbConn)
	usersGrpcService := service.NewAccountHandler(usersStorage, jwtManager, cr)

	secretTypeStorage := postgres.NewCategoryStore(dbConn)
	secretTypeGrpcService := service.NewCategoryHandler(secretTypeStorage)

	secretStorage := postgres.NewDataVaultStore(dbConn)
	secretGrpcService := service.NewDataVaultHandler(secretStorage)

	jwtAuthMiddleware := auth.NewJwtMiddleware(jwtManager, cr).Auth

	gRPCServer := server.NewGrpcServer(
		server.WithServerConfig(cfg),
		server.WithLogger(log),
		server.WithServices(usersGrpcService, secretTypeGrpcService, secretGrpcService),
		server.WithStreamInterceptors(
			grpczap.StreamServerInterceptor(log),
			grpcauth.StreamServerInterceptor(jwtAuthMiddleware),
			grpcrecovery.StreamServerInterceptor(),
		),
		server.WithUnaryInterceptors(
			grpczap.UnaryServerInterceptor(log),
			grpcauth.UnaryServerInterceptor(jwtAuthMiddleware),
			grpcrecovery.UnaryServerInterceptor(),
		),
	)

	return &App{
		DB:     dbConn,
		Server: gRPCServer,
		Logger: log,
	}, nil
}
