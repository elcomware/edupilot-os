package students

import "github.com/elcomware/edupilot-os/platform/catalog"

func Descriptor() catalog.Descriptor {
	return catalog.Descriptor{Key: "students", Group: "lifecycle", Title: catalog.Title{EN: "Students", FR: "Élèves"}, Capabilities: []string{"list", "create", "update", "archive"}}
}
