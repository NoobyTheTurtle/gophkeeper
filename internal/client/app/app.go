package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"

	"github.com/smanhack/gophkeeper/internal/client/config"
	"github.com/smanhack/gophkeeper/internal/client/interceptor"
	"github.com/smanhack/gophkeeper/internal/client/service"
	"github.com/smanhack/gophkeeper/internal/client/storage"
	pb "github.com/smanhack/gophkeeper/pkg/api"
	"github.com/smanhack/gophkeeper/pkg/cert"
	"github.com/smanhack/gophkeeper/pkg/crypt"
)

type App struct {
	DataVaultService *service.DataVaultClientService
	CategoryService  *service.CategoryClientService
	AccountService   *service.AccountClientService

	Storage *storage.MemoryStorage
	Syncer  *storage.Sync
	Cron    *cron.Cron

	jwtToken string
	mu       sync.RWMutex
}

// NewApp - creates Client application.
func NewApp() (*App, error) {
	cfg := config.NewConfig()

	tlsCredential, err := cert.NewSSLConfigService().LoadClientCertificate(cfg)
	if err != nil {
		return nil, fmt.Errorf("error in creating tls creds: %w", err)
	}

	protectedRoutes := map[string]bool{
		"/proto.Account/Remove":                true,
		"/proto.DataCategory/ListCategories":   true,
		"/proto.DataVault/QueryDataByCategory": true,
		"/proto.DataVault/StoreData":           true,
		"/proto.DataVault/FetchData":           true,
		"/proto.DataVault/RemoveData":          true,
		"/proto.DataVault/UpdateData":          true,
	}

	app := &App{}

	intercept := interceptor.NewAuthInterceptor(protectedRoutes, app)

	conn, errConn := grpc.NewClient(":"+cfg.Port,
		grpc.WithTransportCredentials(tlsCredential),
		grpc.WithUnaryInterceptor(intercept.Unary()),
	)
	if errConn != nil {
		return nil, fmt.Errorf("error in creating grpc con:%w", errConn)
	}

	dataVaultClient := pb.NewDataVaultClient(conn)
	accountClient := pb.NewAccountClient(conn)
	categoryClient := pb.NewDataCategoryClient(conn)

	cr, errCr := crypt.NewCrypt()
	if errCr != nil {
		return nil, fmt.Errorf("could create crypt")
	}

	memoryStorage := storage.NewMemoryStorage()
	syn := storage.NewSync(memoryStorage, dataVaultClient, cr)

	dataVaultClientService := service.NewDataVaultClientService(dataVaultClient, memoryStorage, cr, syn)
	accountClientService := service.NewAccountClientService(accountClient)
	categoryClientService := service.NewCategoryClientService(categoryClient)

	app.DataVaultService = dataVaultClientService
	app.CategoryService = categoryClientService
	app.AccountService = accountClientService
	app.Storage = memoryStorage
	app.Syncer = syn

	c := cron.New()
	_, err = c.AddFunc("* * * * *", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if app.GetToken() != "" {
			syn.SyncAll(ctx)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add cron job: %w", err)
	}

	app.Cron = c
	return app, nil
}

func (app *App) SetToken(token string) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.jwtToken = token
}

func (app *App) GetToken() string {
	app.mu.RLock()
	defer app.mu.RUnlock()
	return app.jwtToken
}
