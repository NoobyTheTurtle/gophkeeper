package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/server/middleware/auth"
	"github.com/smanhack/gophkeeper/internal/server/model"
	"github.com/smanhack/gophkeeper/internal/server/storage"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

type dataVaultHandler struct {
	pb.UnimplementedDataVaultServer

	storage storage.DataVaultServerStorage
}

func NewDataVaultHandler(s storage.DataVaultServerStorage) *dataVaultHandler {
	return &dataVaultHandler{
		storage: s,
	}
}

func (s *dataVaultHandler) RegisterService(r grpc.ServiceRegistrar) {
	pb.RegisterDataVaultServer(r, s)
}

func (s *dataVaultHandler) StoreData(ctx context.Context, in *pb.StoreDataRequest) (*pb.StoreDataResponse, error) {
	tok := ctx.Value(auth.JwtTokenCtx{}).(string)

	dataRecord := model.DataRecord{
		AccountID:  uuid.MustParse(tok),
		CategoryID: int(in.Type),
		Name:       in.Name,
		Payload:    in.Payload,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	m, err := s.storage.StoreData(ctx, dataRecord)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var deletedAt pb.NullableDeletedAt
	if m.DeletedAt != nil {
		deletedAt = pb.NullableDeletedAt{Kind: &pb.NullableDeletedAt_Data{Data: timestamppb.New(*m.DeletedAt)}}
	} else {
		deletedAt = pb.NullableDeletedAt{Kind: nil}
	}

	return &pb.StoreDataResponse{
		Id:        uint32(m.ID),
		Name:      m.Name,
		Type:      uint32(m.CategoryID),
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
		DeletedAt: &deletedAt,
	}, nil
}

func (s *dataVaultHandler) FetchData(ctx context.Context, in *pb.FetchDataRequest) (*pb.FetchDataResponse, error) {
	tok := ctx.Value(auth.JwtTokenCtx{}).(string)

	dataRecord := model.DataRecord{
		AccountID: uuid.MustParse(tok),
		ID:        int(in.Id),
	}

	m, err := s.storage.FetchData(ctx, dataRecord)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	var deletedAt pb.NullableDeletedAt
	if m.DeletedAt != nil {
		deletedAt = pb.NullableDeletedAt{Kind: &pb.NullableDeletedAt_Data{Data: timestamppb.New(*m.DeletedAt)}}
	} else {
		deletedAt = pb.NullableDeletedAt{Kind: nil}
	}

	return &pb.FetchDataResponse{
		Id:        uint32(m.ID),
		Name:      m.Name,
		Type:      uint32(m.CategoryID),
		Payload:   m.Payload,
		CreatedAt: timestamppb.New(m.CreatedAt),
		UpdatedAt: timestamppb.New(m.UpdatedAt),
		DeletedAt: &deletedAt,
	}, nil
}

func (s *dataVaultHandler) RemoveData(ctx context.Context, in *pb.RemoveDataRequest) (*pb.RemoveDataResponse, error) {
	tok := ctx.Value(auth.JwtTokenCtx{}).(string)

	dataRecord := model.DataRecord{
		AccountID: uuid.MustParse(tok),
		ID:        int(in.Id),
	}

	_, err := s.storage.RemoveData(ctx, dataRecord)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &pb.RemoveDataResponse{}, nil
}

func (s *dataVaultHandler) UpdateData(ctx context.Context, in *pb.UpdateDataRequest) (*pb.UpdateDataResponse, error) {
	token := ctx.Value(auth.JwtTokenCtx{}).(string)
	dataRecord := model.DataRecord{
		ID:         int(in.Id),
		AccountID:  uuid.MustParse(token),
		Name:       in.Name,
		CategoryID: int(in.Type),
		Payload:    in.Payload,
		UpdatedAt:  in.UpdatedAt.AsTime(),
	}

	updatedDataRecord, err := s.storage.UpdateData(ctx, dataRecord, in.IsForce)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, err.Error())
		}
		if errors.Is(err, errorx.ErrUpdatedAtDoesntMatch) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}

		return nil, status.Error(codes.Internal, err.Error())
	}

	var deletedAt pb.NullableDeletedAt
	if updatedDataRecord.DeletedAt != nil {
		deletedAt = pb.NullableDeletedAt{
			Kind: &pb.NullableDeletedAt_Data{
				Data: timestamppb.New(*updatedDataRecord.DeletedAt),
			},
		}
	} else {
		deletedAt = pb.NullableDeletedAt{Kind: nil}
	}

	return &pb.UpdateDataResponse{
		Id:        uint32(updatedDataRecord.ID),
		Name:      updatedDataRecord.Name,
		Type:      uint32(updatedDataRecord.CategoryID),
		CreatedAt: timestamppb.New(updatedDataRecord.CreatedAt),
		UpdatedAt: timestamppb.New(updatedDataRecord.UpdatedAt),
		DeletedAt: &deletedAt,
	}, nil
}

func (s *dataVaultHandler) QueryDataByCategory(
	ctx context.Context, in *pb.QueryDataByCategoryRequest,
) (*pb.QueryDataByCategoryResponse, error) {
	token := ctx.Value(auth.JwtTokenCtx{}).(string)

	userId, errParse := uuid.Parse(token)
	if errParse != nil {
		return nil, status.Error(codes.Internal, errParse.Error())
	}

	user := model.Account{ID: &userId}
	dataCategory := model.DataCategory{ID: uint(in.TypeId)}

	dataRecords, err := s.storage.QueryDataByCategory(ctx, dataCategory, user)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var castedDataRecords []*pb.DataRecord

	for _, val := range dataRecords {
		var deletedAt pb.NullableDeletedAt
		if val.DeletedAt != nil {
			deletedAt = pb.NullableDeletedAt{Kind: &pb.NullableDeletedAt_Data{Data: timestamppb.New(*val.DeletedAt)}}
		} else {
			deletedAt = pb.NullableDeletedAt{Kind: nil}
		}

		castedDataRecords = append(castedDataRecords, &pb.DataRecord{
			Id:        uint32(val.ID),
			UserId:    val.AccountID.String(),
			TypeId:    uint32(val.CategoryID),
			Name:      val.Name,
			Payload:   val.Payload,
			CreatedAt: timestamppb.New(val.CreatedAt),
			UpdatedAt: timestamppb.New(val.UpdatedAt),
			DeletedAt: &deletedAt,
		})
	}

	return &pb.QueryDataByCategoryResponse{SecretLists: castedDataRecords}, nil
}
