package service

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/server/storage"
)

type categoryHandler struct {
	pb.UnimplementedDataCategoryServer

	storage storage.CategoryServerStorage
}

func NewCategoryHandler(s storage.CategoryServerStorage) *categoryHandler {
	return &categoryHandler{storage: s}
}

func (s *categoryHandler) RegisterService(r grpc.ServiceRegistrar) {
	pb.RegisterDataCategoryServer(r, s)
}

func (s *categoryHandler) ListCategories(
	ctx context.Context, in *pb.CategoryListRequest,
) (*pb.CategoryListResponse, error) {
	list, err := s.storage.ListCategories(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	resp := &pb.CategoryListResponse{}
	for _, category := range list {
		resp.Categories = append(resp.Categories, &pb.Category{
			Id:    uint32(category.ID),
			Title: category.Name,
		})
	}

	return resp, nil
}
