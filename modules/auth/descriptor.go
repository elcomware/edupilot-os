package auth

import "github.com/elcomware/edupilot/modules/core/catalog"

func Descriptor() catalog.Descriptor {
	return catalog.Descriptor{Key: "auth", Group: "platform", Title: catalog.Title{EN: "Accounts and access", FR: "Comptes et accès"}, Capabilities: []string{"self"}}
}
