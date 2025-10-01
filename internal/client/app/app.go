package app

import (
	"context"
	"fmt"

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
}

// NewApp - creates Client application.
func NewApp(ctx context.Context) (*App, context.CancelFunc, error) {
	ctx, cancel := context.WithCancel(ctx)

	cfg := config.NewConfig()

	tlsCredential, err := cert.NewSSLConfigService().LoadClientCertificate(cfg)
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("error in creating tls creds: %w", err)
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
	intercept := interceptor.NewAuthInterceptor(protectedRoutes)

	conn, errConn := grpc.NewClient(":"+cfg.Port,
		grpc.WithTransportCredentials(tlsCredential),
		grpc.WithUnaryInterceptor(intercept.Unary()),
	)
	if errConn != nil {
		cancel()
		return nil, nil, fmt.Errorf("error in creating grpc con:%w", errConn)
	}

	dataVaultClient := pb.NewDataVaultClient(conn)
	accountClient := pb.NewAccountClient(conn)
	categoryClient := pb.NewDataCategoryClient(conn)

	cr, errCr := crypt.NewCrypt()
	if errCr != nil {
		cancel()
		return nil, nil, fmt.Errorf("could create crypt")
	}

	memoryStorage := storage.NewMemoryStorage()
	syn := storage.NewSync(memoryStorage, dataVaultClient, cr)

	dataVaultClientService := service.NewDataVaultClientService(dataVaultClient, memoryStorage, cr, syn)
	accountClientService := service.NewAccountClientService(accountClient)
	categoryClientService := service.NewCategoryClientService(categoryClient)

	c := cron.New()
	_, err = c.AddFunc("* * * * *", func() { syn.SyncAll(ctx) })
	if err != nil {
		cancel()
		return nil, nil, fmt.Errorf("failed to add cron job: %w", err)
	}

	return &App{
		DataVaultService: dataVaultClientService,
		CategoryService:  categoryClientService,
		AccountService:   accountClientService,
		Storage:          memoryStorage,
		Syncer:           syn,
		Cron:             c,
	}, cancel, nil
}
