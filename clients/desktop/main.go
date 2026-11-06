// The desktop binary embeds the exact web distribution used by Goravel.
// Prepare assets using scripts/build.mjs before invoking the Go toolchain.
package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"edupilot.local/desktop/internal/gateway"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const productName = "EduPilot OS"

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	root, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		log.Fatal(err)
	}
	origin, err := gateway.Origin(os.Getenv("EDUPILOT_API_ORIGIN"))
	if err != nil {
		log.Fatal(err)
	}
	api, closeIdle := gateway.New(origin)
	defer closeIdle()
	app := application.New(application.Options{
		Name:        productName,
		Description: "EduPilot OS school workspace",
		// Do not allow a development-server environment variable to swap the
		// application supplying credentials. Serve only the embedded bundle.
		Assets: application.AssetOptions{
			Handler: http.FileServer(http.FS(root)),
			Middleware: func(next http.Handler) http.Handler {
				return gateway.Mount(next, api)
			},
		},
		Mac: application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: productName, Width: 1280, Height: 830,
		MinWidth: 720, MinHeight: 530, URL: "/",
	})
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
