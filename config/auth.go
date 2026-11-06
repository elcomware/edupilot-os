package config

import "github.com/elcomware/edupilot/app/facades"

func init() {
	facades.Config().Add("auth", map[string]any{
		"defaults":  map[string]any{"guard": "school"},
		"guards":    map[string]any{"school": map[string]any{"driver": "school-bearer", "provider": "users"}},
		"providers": map[string]any{"users": map[string]any{"driver": "orm"}},
	})
}
