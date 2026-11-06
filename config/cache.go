package config

import (
	"github.com/elcomware/edupilot/app/facades"
)

func init() {
	config := facades.Config()
	config.Add("cache", map[string]any{
		// Default Cache Store
		//
		// This option controls the default cache connection that gets used while
		// using this caching library. This connection is used when another is
		// not explicitly specified when executing a given caching function.
		"default": "memory",

		// Cache Stores
		//
		// Here you may define all the cache "stores" for your application as
		// well as their drivers. You may even define multiple stores for the
		// same cache driver to group types of items stored in your caches.
		// Available Drivers: "memory", "custom"
		"stores": map[string]any{
			"memory": map[string]any{
				"driver": "memory",
			},
		},

		"prefix": config.Env("CACHE_PREFIX", "edupilot"),
	})
}
