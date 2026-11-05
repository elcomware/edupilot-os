package config

import "github.com/elcomware/edupilot-os/app/facades"

func init() {
	config := facades.Config()
	config.Add("edupilot", map[string]any{"enabled_modules": config.Env("EDUPILOT_ENABLED_MODULES", "")})
}
