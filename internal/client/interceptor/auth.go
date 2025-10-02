package interceptor

import (
	"context"
	"errors"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type AuthInterceptor struct {
	protectedMethods map[string]bool
	tokenProvider    TokenProvider
}

// NewAuthInterceptor - returns an Auth interceptor
func NewAuthInterceptor(prMethods map[string]bool, tokenProvider TokenProvider) *AuthInterceptor {
	return &AuthInterceptor{
		protectedMethods: prMethods,
		tokenProvider:    tokenProvider,
	}
}

// Unary returns a client interceptor to authenticate unary RPC
func (a *AuthInterceptor) Unary() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		if a.protectedMethods[method] {
			token := a.tokenProvider.GetToken()

			if token == "" {
				return errors.New("you have to be authorized via login first")
			}

			ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
