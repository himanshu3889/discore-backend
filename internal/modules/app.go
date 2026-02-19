package app

import (
	"google.golang.org/grpc"

	pb "github.com/himanshu3889/discore-backend/protos/auth"
)

type modulesAppState struct {
	AuthClient pb.AuthServiceClient
}

var (
	ModulesApp *modulesAppState
)

// SetGlobal initializes the variable
func SetState(conn *grpc.ClientConn) {
	authClient := pb.NewAuthServiceClient(conn)
	ModulesApp = &modulesAppState{AuthClient: authClient}
}
