package service

import (
	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/client/model"
)

type CategoryClientService struct {
	glCtx  *model.GlobalContext
	client pb.DataCategoryClient
}

// NewCategoryClientService - создает новый CategoryClientService.
func NewCategoryClientService(glCtx *model.GlobalContext, client pb.DataCategoryClient) *CategoryClientService {
	return &CategoryClientService{
		glCtx:  glCtx,
		client: client,
	}
}

func (s *CategoryClientService) List() (*pb.CategoryListResponse, error) {
	request := &pb.CategoryListRequest{}

	result, err := s.client.ListCategories(s.glCtx.Ctx, request)
	if err != nil {
		return nil, err
	}

	return result, nil
}
