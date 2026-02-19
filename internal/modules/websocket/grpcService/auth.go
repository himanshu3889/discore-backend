package grpcService

import (
	"context"
	"time"

	app "github.com/himanshu3889/discore-backend/internal/modules"
	pb "github.com/himanshu3889/discore-backend/protos/auth"
)

func ValidateToken(token string) (*pb.ValidateAccessTokenResponse, error) {

	client := app.ModulesApp.AuthClient

	// Timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create the Request Object
	req := &pb.ValidateAccessTokenRequest{
		Token: token,
	}

	// Call the RPC method
	return client.ValidateAccessToken(ctx, req)

}
