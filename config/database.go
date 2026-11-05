package config

import (
	"github.com/elcomware/edupilot-os/app/facades"
	"github.com/elcomware/edupilot-os/platform/database"
)

func init() {
	settings, err := database.ConfiguredSettings()
	if err != nil {
		panic(err)
	}
	facades.Config().Add("database", map[string]any{
		"default":        settings.Control.Name,
		"connections":    settings.Connections(),
		"pool":           map[string]any{"max_idle_conns": 5, "max_open_conns": 12, "conn_max_idletime": 300, "conn_max_lifetime": 3600},
		"slow_threshold": 200,
		"migrations":     map[string]any{"table": "migrations"},
	})
}
