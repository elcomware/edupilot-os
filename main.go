package main

import (
	"github.com/elcomware/edupilot/bootstrap"
)

func main() {
	app := bootstrap.Boot()

	app.Start()
}
