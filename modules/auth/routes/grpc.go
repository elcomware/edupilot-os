package routes

import (
	authv1 "github.com/elcomware/edupilot/modules/auth/proto/v1"
	"google.golang.org/grpc"
)

func Grpc(server grpc.ServiceRegistrar, controller authv1.AuthServer) {
	authv1.RegisterAuthServer(server, controller)
}
