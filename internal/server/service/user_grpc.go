package service

import (
	"context"
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smanhack/gophkeeper/internal/server/middleware/auth"
	"github.com/smanhack/gophkeeper/internal/server/model"
	pb "github.com/smanhack/gophkeeper/pkg/api"
	"github.com/smanhack/gophkeeper/pkg/errorx"
)

type accountHandler struct {
	pb.UnimplementedAccountServer

	storage    AccountServerStorage
	jwtManager JWTManager
	crypter    Crypter
}

func NewAccountHandler(s AccountServerStorage, m JWTManager, c Crypter) *accountHandler {
	return &accountHandler{
		storage:    s,
		jwtManager: m,
		crypter:    c,
	}
}

func (u *accountHandler) RegisterService(r grpc.ServiceRegistrar) {
	pb.RegisterAccountServer(r, u)
}

func (u *accountHandler) SignUp(ctx context.Context, in *pb.SignUpRequest) (*pb.SignUpResponse, error) {
	m := model.Account{Username: in.Username, Credential: in.Credential}

	validate := validator.New()
	if errV := validate.Struct(m); errV != nil {
		return nil, status.Error(codes.InvalidArgument, errV.Error())
	}

	accountModel, err := u.storage.StoreAccount(ctx, m)

	if errors.Is(err, errorx.ErrConflict) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, errToken := u.jwtManager.Issue(accountModel.ID.String())
	if errToken != nil {
		return nil, status.Error(codes.Internal, errToken.Error())
	}

	return &pb.SignUpResponse{Token: u.crypter.Encode(token)}, nil
}

func (u *accountHandler) Authenticate(ctx context.Context, in *pb.AuthRequest) (*pb.AuthResponse, error) {
	accountModel, err := u.storage.FindByCredentials(ctx, model.Account{Username: in.Username, Credential: in.Credential})

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	token, errToken := u.jwtManager.Issue(accountModel.ID.String())
	if errToken != nil {
		return nil, status.Error(codes.Internal, errToken.Error())
	}

	return &pb.AuthResponse{Token: u.crypter.Encode(token)}, nil
}

func (u *accountHandler) Remove(ctx context.Context, in *pb.RemoveRequest) (*pb.RemoveResponse, error) {
	token := ctx.Value(auth.JwtTokenCtx{}).(string)

	uid, err := uuid.Parse(token)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	_, errDelete := u.storage.RemoveAccount(ctx, model.Account{ID: &uid})
	if errDelete != nil {
		if errors.Is(errDelete, pgx.ErrNoRows) {
			return nil, status.Error(codes.NotFound, errDelete.Error())
		}
		return nil, status.Error(codes.Internal, errDelete.Error())
	}

	return &pb.RemoveResponse{}, nil
}
