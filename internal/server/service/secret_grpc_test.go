package service

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/google/uuid"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smanhack/gophkeeper/internal/server/middleware/auth"
	"github.com/smanhack/gophkeeper/internal/server/model"
	servicemock "github.com/smanhack/gophkeeper/internal/server/service/mock"
	pb "github.com/smanhack/gophkeeper/pkg/api"
)

var (
	loc, _ = time.LoadLocation("UTC")
	now    = time.Now().In(loc)
)

func TestDataVaultHandler_RegisterService(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	dataVaultMock := servicemock.NewMockDataVaultServerStorage(ctl)

	tests := []struct {
		name string
	}{
		{
			name: "Registrar can be called without errors",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewDataVaultHandler(dataVaultMock)

			server := grpc.NewServer()

			s.RegisterService(server)
		})
	}
}

func TestDataVaultHandler_StoreData(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := dataVaultTestClient(t, ctl, uid)
	defer close(done)

	_, err := client.StoreData(ctx, &pb.StoreDataRequest{})
	assert.NoError(t, err)
}

func TestDataVaultHandler_FetchData(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := dataVaultTestClient(t, ctl, uid)
	defer close(done)

	_, err := client.FetchData(ctx, &pb.FetchDataRequest{Id: 1})
	assert.NoError(t, err)

	_, err = client.FetchData(ctx, &pb.FetchDataRequest{Id: 0})
	assert.Error(t, err)
}

func TestDataVaultHandler_RemoveData(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := dataVaultTestClient(t, ctl, uid)
	defer close(done)

	_, err := client.RemoveData(ctx, &pb.RemoveDataRequest{Id: 1})
	assert.NoError(t, err)

	_, err = client.RemoveData(ctx, &pb.RemoveDataRequest{Id: 0})
	assert.Error(t, err)
}

func TestDataVaultHandler_UpdateData(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := dataVaultTestClient(t, ctl, uid)
	defer close(done)

	_, err := client.UpdateData(ctx, &pb.UpdateDataRequest{Id: 1, IsForce: true, UpdatedAt: timestamppb.New(now)})
	assert.NoError(t, err)

	_, err = client.UpdateData(ctx, &pb.UpdateDataRequest{Id: 0, IsForce: true, UpdatedAt: timestamppb.New(now)})
	assert.Error(t, err)
}

func TestDataVaultHandler_QueryDataByCategory(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := dataVaultTestClient(t, ctl, uid)
	defer close(done)

	res, err := client.QueryDataByCategory(ctx, &pb.QueryDataByCategoryRequest{})
	assert.NoError(t, err)
	assert.Len(t, res.SecretLists, 2)
}

func dataVaultTestClient(t *testing.T, ctl *gomock.Controller, uid uuid.UUID) (pb.DataVaultClient, chan<- struct{}) {
	done := make(chan struct{})

	dataVaultStorageMock := servicemock.NewMockDataVaultServerStorage(ctl)

	dataVaultStorageMock.EXPECT().StoreData(gomock.Any(), gomock.Any()).AnyTimes().Return(model.DataRecord{}, nil)

	dataVaultStorageMock.EXPECT().
		FetchData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 1, AccountID: uid})).
		AnyTimes().
		Return(model.DataRecord{}, nil)
	dataVaultStorageMock.EXPECT().
		FetchData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 0, AccountID: uid})).
		AnyTimes().
		Return(model.DataRecord{}, errors.New("test"))

	dataVaultStorageMock.EXPECT().
		RemoveData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 1, AccountID: uid})).
		AnyTimes().
		Return(model.DataRecord{}, nil)
	dataVaultStorageMock.EXPECT().
		RemoveData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 0, AccountID: uid})).
		AnyTimes().
		Return(model.DataRecord{}, errors.New("test"))

	dataVaultStorageMock.EXPECT().
		UpdateData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 1, AccountID: uid, UpdatedAt: now}), gomock.Eq(true)).
		AnyTimes().
		Return(model.DataRecord{}, nil)
	dataVaultStorageMock.EXPECT().
		UpdateData(gomock.Any(), gomock.Eq(model.DataRecord{ID: 0, AccountID: uid, UpdatedAt: now}), gomock.Eq(true)).
		AnyTimes().
		Return(model.DataRecord{}, errors.New("test"))

	dataVaultStorageMock.EXPECT().QueryDataByCategory(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().
		Return([]model.DataRecord{
			{ID: 1},
			{ID: 2},
		}, nil)

	jwtM := servicemock.NewMockJWTManager(ctl)
	jwtM.EXPECT().Issue(uid.String()).AnyTimes().Return("token", nil)
	jwtM.EXPECT().Decode(gomock.Any()).AnyTimes().Return(uid.String(), nil)

	cryptM := servicemock.NewMockCrypter(ctl)
	cryptM.EXPECT().Encode(gomock.Any()).AnyTimes().Return("token")
	cryptM.EXPECT().Decode(gomock.Any()).AnyTimes().Return(uid.String(), nil)

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	dataVaultRpc := NewDataVaultHandler(dataVaultStorageMock)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpcauth.UnaryServerInterceptor(auth.NewJwtMiddleware(jwtM, cryptM).Auth),
			)),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpcauth.StreamServerInterceptor(auth.NewJwtMiddleware(jwtM, cryptM).Auth),
			)),
	)

	pb.RegisterDataVaultServer(server, dataVaultRpc)

	go func() {
		if err = server.Serve(l); err != nil && err != grpc.ErrServerStopped {
			panic(err)
		}
	}()

	conn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		<-done
		server.GracefulStop()
		_ = conn.Close() // Explicitly ignore close error in test cleanup
	}()

	client := pb.NewDataVaultClient(conn)

	return client, done
}
