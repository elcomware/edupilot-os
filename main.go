package main

import (
	"github.com/elcomware/edupilot-os/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	app.Start()
}
