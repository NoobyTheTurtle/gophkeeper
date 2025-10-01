package service

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/smanhack/gophkeeper/internal/client/model"
	pb "github.com/smanhack/gophkeeper/pkg/api"
)

type AccountClientService struct {
	client pb.AccountClient
}

func NewAccountClientService(client pb.AccountClient) *AccountClientService {
	return &AccountClientService{
		client: client,
	}
}

func (u *AccountClientService) Authenticate(ctx context.Context, account model.Account) (context.Context, error) {
	request := &pb.AuthRequest{
		Username:   account.Username,
		Credential: account.Credential,
	}

	result, err := u.client.Authenticate(ctx, request)
	if err != nil {
		return ctx, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+result.Token)

	return ctx, nil
}

// SignUp - creates a new user on server. On successful creation adds authorization token to metadata in context.
func (u *AccountClientService) SignUp(ctx context.Context, account model.Account) (context.Context, error) {
	result, err := u.client.SignUp(ctx, &pb.SignUpRequest{Username: account.Username, Credential: account.Credential})
	if err != nil {
		return ctx, err
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+result.Token)

	return ctx, nil
}

// Remove - removes a user from server. On successful removal, removes authorization token from metadata in context.
func (u *AccountClientService) Remove(ctx context.Context) (context.Context, error) {
	_, err := u.client.Remove(ctx, &pb.RemoveRequest{})
	if err != nil {
		return ctx, err
	}

	ctx = metadata.NewOutgoingContext(ctx, metadata.MD{})

	return ctx, nil
}

// Logout - removes authorization token from metadata in context.
func (u *AccountClientService) Logout(ctx context.Context) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.MD{})
}
