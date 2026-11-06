package routes

import (
	"github.com/elcomware/edupilot/app/facades"
	"github.com/elcomware/edupilot/modules/auth/app/http/controllers"
	authroutes "github.com/elcomware/edupilot/modules/auth/routes"
)

func Grpc(auth *controllers.AuthController) {
	authroutes.Grpc(facades.Grpc().Server(), auth)
}
