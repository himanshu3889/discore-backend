package authGrpc

import (
	"context"

	"github.com/himanshu3889/discore-backend/configs"
	"github.com/himanshu3889/discore-backend/internal/gateway/authenticationService/jwtAuthentication"
	authpb "github.com/himanshu3889/discore-backend/protos/auth"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AuthServer struct {
	authpb.UnimplementedAuthServiceServer
}

func (s *AuthServer) ValidateAccessToken(
	ctx context.Context,
	req *authpb.ValidateAccessTokenRequest,
) (*authpb.ValidateAccessTokenResponse, error) {

	if req.Token == "" {
		return nil, status.Error(codes.InvalidArgument, "token is required")
	}

	token, err := jwt.ParseWithClaims(req.Token, &jwtAuthentication.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(configs.Config.JWT_SECRET), nil
	})

	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	if !token.Valid {
		return nil, status.Error(codes.Unauthenticated, "token not valid")
	}

	claims, claimsOk := token.Claims.(*jwtAuthentication.JwtClaims)
	if !claimsOk {
		return nil, status.Error(codes.Internal, "failed to parse claims")
	}

	return &authpb.ValidateAccessTokenResponse{
		UserID: int64(claims.UserId),
		Email:  claims.Email,
	}, nil
}
