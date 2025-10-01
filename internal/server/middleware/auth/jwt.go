package auth

import (
	"context"

	grpcauth "github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type (
	JwtTokenCtx   struct{}
	JwtMiddleware struct {
		jwtManager         JWTManager
		unProtectedMethods []string
		crypter            Crypter
	}
)

func NewJwtMiddleware(j JWTManager, c Crypter) *JwtMiddleware {
	return &JwtMiddleware{
		jwtManager:         j,
		crypter:            c,
		unProtectedMethods: []string{"/proto.Account/SignUp", "/proto.Account/Authenticate"},
	}
}

func (a *JwtMiddleware) Auth(ctx context.Context) (context.Context, error) {
	if a.isSkippingCurrentRoute(ctx) {
		return ctx, nil
	}

	encToken, err := grpcauth.AuthFromMD(ctx, "bearer")
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "bearer could not be retrieved: %v", err)
	}

	token, errCrypter := a.crypter.Decode(encToken)
	if errCrypter != nil {
		return nil, status.Errorf(codes.Unauthenticated, "crypter decoding error: %v", errCrypter)
	}

	decodedToken, errDecode := a.jwtManager.Decode(token)
	if errDecode != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid auth token: %v", errDecode)
	}

	newCtx := context.WithValue(ctx, JwtTokenCtx{}, decodedToken)

	return newCtx, nil
}

func (a *JwtMiddleware) isSkippingCurrentRoute(ctx context.Context) bool {
	isSkipping := false

	calledMethod, _ := grpc.Method(ctx)
	for _, method := range a.unProtectedMethods {
		if calledMethod == method {
			isSkipping = true
		}
	}

	return isSkipping
}
