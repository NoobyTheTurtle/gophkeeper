package service

import (
	"google.golang.org/grpc/metadata"

	pb "github.com/smanhack/gophkeeper/api/proto"
	"github.com/smanhack/gophkeeper/internal/client/model"
)

type AccountClientService struct {
	glCtx  *model.GlobalContext
	client pb.AccountClient
}

func NewAccountClientService(glCtx *model.GlobalContext, client pb.AccountClient) *AccountClientService {
	return &AccountClientService{
		glCtx:  glCtx,
		client: client,
	}
}

func (u *AccountClientService) Authenticate(account model.Account) error {
	request := &pb.AuthRequest{
		Username:   account.Username,
		Credential: account.Credential,
	}

	result, err := u.client.Authenticate(u.glCtx.Ctx, request)
	if err != nil {
		return err
	}

	u.glCtx.Ctx = metadata.AppendToOutgoingContext(u.glCtx.Ctx, "authorization", "Bearer "+result.Token)

	return nil
}

// SignUp - creates a new user on server. On successful creation adds authorization token to metadata in global
// shared context.
func (u *AccountClientService) SignUp(account model.Account) error {
	result, err := u.client.SignUp(u.glCtx.Ctx, &pb.SignUpRequest{Username: account.Username, Credential: account.Credential})
	if err != nil {
		return err
	}

	u.glCtx.Ctx = metadata.AppendToOutgoingContext(u.glCtx.Ctx, "authorization", "Bearer "+result.Token)

	return nil
}

// Remove - removes a user from server. On successful removal, removes authorization token from metadata in global
// shared context.
func (u *AccountClientService) Remove() error {
	_, err := u.client.Remove(u.glCtx.Ctx, &pb.RemoveRequest{})
	if err != nil {
		return err
	}

	u.glCtx.Ctx = metadata.NewOutgoingContext(u.glCtx.Ctx, metadata.MD{})

	return nil
}

// Logout - removes authorization token from metadata in global shared context.
func (u *AccountClientService) Logout() {
	u.glCtx.Ctx = metadata.NewOutgoingContext(u.glCtx.Ctx, metadata.MD{})
}
