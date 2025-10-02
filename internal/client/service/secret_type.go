package service

import (
	"context"

	pb "github.com/smanhack/gophkeeper/pkg/api"
)

type CategoryClientService struct {
	client pb.DataCategoryClient
}

// NewCategoryClientService - создает новый CategoryClientService.
func NewCategoryClientService(client pb.DataCategoryClient) *CategoryClientService {
	return &CategoryClientService{
		client: client,
	}
}

func (s *CategoryClientService) List(ctx context.Context) (*pb.CategoryListResponse, error) {
	request := &pb.CategoryListRequest{}

	result, err := s.client.ListCategories(ctx, request)
	if err != nil {
		return nil, err
	}

	return result, nil
}
