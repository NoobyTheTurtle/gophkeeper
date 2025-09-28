package service

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/server/middleware/auth"
	"github.com/smanhack/gophkeeper/internal/server/model"
	storagemock "github.com/smanhack/gophkeeper/internal/server/storage/mock"
	cryptmock "github.com/smanhack/gophkeeper/pkg/crypt/mock"
	"github.com/smanhack/gophkeeper/pkg/errorx"
	jwtmock "github.com/smanhack/gophkeeper/pkg/jwt/mock"
)

func Test_accountHandler_RegisterService(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	accountMock := storagemock.NewMockAccountServerStorage(ctl)
	jwtMock := jwtmock.NewMockManager(ctl)
	cryptMock := cryptmock.NewMockCrypter(ctl)

	tests := []struct {
		name string
	}{
		{
			name: "Registrar can be called without errors",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewAccountHandler(accountMock, jwtMock, cryptMock)

			server := grpc.NewServer()

			s.RegisterService(server)
		})
	}
}

func Test_accountHandler_SignUp(t *testing.T) {
	uid := uuid.New()
	ctx := context.Background()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := accountTestClientNoAuth(t, ctl, uid)
	defer close(done)

	_, err := client.SignUp(ctx, &pb.SignUpRequest{
		Username:   "RegConflict",
		Credential: "pass",
	})
	assert.Error(t, err)

	_, err = client.SignUp(ctx, &pb.SignUpRequest{
		Username:   "RegOk",
		Credential: "pass",
	})
	assert.NoError(t, err)
}

func Test_accountHandler_Authenticate(t *testing.T) {
	uid := uuid.New()
	ctx := context.Background()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := accountTestClientNoAuth(t, ctl, uid)
	defer close(done)

	_, err := client.Authenticate(ctx, &pb.AuthRequest{
		Username:   "usernameOk",
		Credential: "pass",
	})
	assert.NoError(t, err)

	_, err = client.Authenticate(ctx, &pb.AuthRequest{
		Username:   "usernameErr",
		Credential: "pass",
	})
	assert.Error(t, err)
}

func Test_accountHandler_Remove(t *testing.T) {
	uid := uuid.New()
	ctx := metadata.AppendToOutgoingContext(context.Background(), "authorization", "Bearer token")

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	client, done := accountTestClientWithAuth(t, ctl, uid)
	defer close(done)

	_, err := client.Remove(ctx, &pb.RemoveRequest{})
	assert.NoError(t, err)
}

func accountTestClientNoAuth(t *testing.T, ctl *gomock.Controller, uid uuid.UUID) (pb.AccountClient, chan<- struct{}) {
	done := make(chan struct{})

	accountStorageMock := storagemock.NewMockAccountServerStorage(ctl)

	accountStorageMock.
		EXPECT().
		FindByCredentials(gomock.Any(), gomock.Eq(model.Account{Username: "usernameOk", Credential: "pass"})).
		AnyTimes().
		Return(model.Account{ID: &uid, Username: "test", Credential: "pass"}, nil)

	accountStorageMock.
		EXPECT().
		FindByCredentials(gomock.Any(), gomock.Eq(model.Account{Username: "usernameErr", Credential: "pass"})).
		AnyTimes().
		Return(model.Account{Username: "test", Credential: "pass"}, errors.New("user login error"))

	accountStorageMock.
		EXPECT().
		StoreAccount(gomock.Any(), gomock.Eq(model.Account{Username: "RegConflict", Credential: "pass"})).
		AnyTimes().
		Return(model.Account{Username: "test", Credential: "pass"}, errorx.ErrConflict)

	accountStorageMock.
		EXPECT().
		StoreAccount(gomock.Any(), gomock.Eq(model.Account{Username: "RegOk", Credential: "pass"})).
		AnyTimes().
		Return(model.Account{ID: &uid, Username: "test1", Credential: "pass"}, nil)

	jwtM := jwtmock.NewMockManager(ctl)
	jwtM.EXPECT().Issue(uid.String()).AnyTimes().Return("token", nil)

	cryptM := cryptmock.NewMockCrypter(ctl)
	cryptM.EXPECT().Encode(gomock.Any()).AnyTimes().Return("token")

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	accountRpc := NewAccountHandler(accountStorageMock, jwtM, cryptM)

	// No auth middleware for SignUp/Authenticate
	server := grpc.NewServer()

	pb.RegisterAccountServer(server, accountRpc)

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

	client := pb.NewAccountClient(conn)

	return client, done
}

func accountTestClientWithAuth(t *testing.T, ctl *gomock.Controller, uid uuid.UUID) (pb.AccountClient, chan<- struct{}) {
	done := make(chan struct{})

	accountStorageMock := storagemock.NewMockAccountServerStorage(ctl)
	accountStorageMock.
		EXPECT().
		RemoveAccount(
			gomock.Any(),
			gomock.Any(),
		).
		AnyTimes().
		Return(model.Account{}, nil)

	jwtM := jwtmock.NewMockManager(ctl)
	jwtM.EXPECT().Issue(uid.String()).AnyTimes().Return("token", nil)
	jwtM.EXPECT().Decode(gomock.Any()).AnyTimes().Return(uid.String(), nil)

	cryptM := cryptmock.NewMockCrypter(ctl)
	cryptM.EXPECT().Encode(gomock.Any()).AnyTimes().Return("token")
	cryptM.EXPECT().Decode(gomock.Any()).AnyTimes().Return(uid.String(), nil)

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	accountRpc := NewAccountHandler(accountStorageMock, jwtM, cryptM)

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

	pb.RegisterAccountServer(server, accountRpc)

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

	client := pb.NewAccountClient(conn)

	return client, done
}
