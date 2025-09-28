package app

import (
	"context"
	"fmt"

	"github.com/robfig/cron/v3"
	"google.golang.org/grpc"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/client/config"
	"github.com/smanhack/gophkeeper/internal/client/interceptor"
	"github.com/smanhack/gophkeeper/internal/client/model"
	"github.com/smanhack/gophkeeper/internal/client/service"
	"github.com/smanhack/gophkeeper/internal/client/storage"
	"github.com/smanhack/gophkeeper/pkg/cert"
	"github.com/smanhack/gophkeeper/pkg/crypt"
)

type App struct {
	Cancel context.CancelFunc

	DataVaultService *service.DataVaultClientService
	CategoryService  *service.CategoryClientService
	AccountService   *service.AccountClientService

	Storage storage.Memorier
	Syncer  storage.Syncer
	Cron    *cron.Cron
}

// NewApp - creates Client application.
func NewApp() (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())
	glCtx := model.GlobalContext{Ctx: ctx, Cancel: cancel}

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
	intercept := interceptor.NewAuthInterceptor(protectedRoutes)

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
	syn := storage.NewSync(memoryStorage, dataVaultClient, &glCtx, cr)

	dataVaultClientService := service.NewDataVaultClientService(&glCtx, dataVaultClient, memoryStorage, cr, syn)
	accountClientService := service.NewAccountClientService(&glCtx, accountClient)
	categoryClientService := service.NewCategoryClientService(&glCtx, categoryClient)

	c := cron.New()
	_, err = c.AddFunc("* * * * *", syn.SyncAll)
	if err != nil {
		return nil, fmt.Errorf("failed to add cron job: %w", err)
	}

	return &App{
		DataVaultService: dataVaultClientService,
		CategoryService:  categoryClientService,
		AccountService:   accountClientService,
		Storage:          memoryStorage,
		Syncer:           syn,
		Cron:             c,
		Cancel:           cancel,
	}, nil
}
