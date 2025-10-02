package service

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smanhack/gophkeeper/internal/server/model"
	servicemock "github.com/smanhack/gophkeeper/internal/server/service/mock"
	pb "github.com/smanhack/gophkeeper/pkg/api"
)

func Test_categoryHandler_ListCategories(t *testing.T) {
	ctx := context.Background()

	ctl := gomock.NewController(t)
	defer ctl.Finish()

	categoryMock := servicemock.NewMockCategoryServerStorage(ctl)

	categoryMock.EXPECT().ListCategories(gomock.Any()).AnyTimes().Return(
		[]model.DataCategory{
			{ID: 1, Name: "Test"},
			{ID: 2, Name: "Test 2"},
		},
		nil,
	)

	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}

	categoryRpc := NewCategoryHandler(categoryMock)

	server := grpc.NewServer()
	defer server.GracefulStop()

	pb.RegisterDataCategoryServer(server, categoryRpc)

	go func() {
		if err = server.Serve(l); err != nil && err != grpc.ErrServerStopped {
			panic(err)
		}
	}()

	conn, err := grpc.NewClient(l.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }() // Explicitly ignore close error in tests

	client := pb.NewDataCategoryClient(conn)

	resp, err := client.ListCategories(ctx, &pb.CategoryListRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Categories, 2)
}

func Test_categoryHandler_RegisterService(t *testing.T) {
	ctl := gomock.NewController(t)
	defer ctl.Finish()

	categoryMock := servicemock.NewMockCategoryServerStorage(ctl)

	tests := []struct {
		name string
	}{
		{
			name: "Registrar can be called without errors",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewCategoryHandler(categoryMock)

			server := grpc.NewServer()

			s.RegisterService(server)
		})
	}
}
