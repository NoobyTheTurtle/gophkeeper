package service

import (
	"context"

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

func (u *AccountClientService) Authenticate(ctx context.Context, account model.Account) (string, error) {
	request := &pb.AuthRequest{
		Username:   account.Username,
		Credential: account.Credential,
	}

	result, err := u.client.Authenticate(ctx, request)
	if err != nil {
		return "", err
	}

	return result.Token, nil
}

func (u *AccountClientService) SignUp(ctx context.Context, account model.Account) (string, error) {
	result, err := u.client.SignUp(ctx, &pb.SignUpRequest{Username: account.Username, Credential: account.Credential})
	if err != nil {
		return "", err
	}

	return result.Token, nil
}

func (u *AccountClientService) Remove(ctx context.Context) error {
	_, err := u.client.Remove(ctx, &pb.RemoveRequest{})
	return err
}

func (u *AccountClientService) Logout() string {
	return ""
}
