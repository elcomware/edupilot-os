package tenancy

import "github.com/elcomware/edupilot/modules/core/catalog"

const Key = "tenancy"

func Descriptor() catalog.Descriptor {
	return catalog.Descriptor{Key: Key, Group: "platform", Title: catalog.Title{EN: "Schools", FR: "Établissements"}, Capabilities: []string{"list", "create", "update"}}
}
