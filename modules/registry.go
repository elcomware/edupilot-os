package modules

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/elcomware/edupilot-os/modules/modulekit"
	"github.com/elcomware/edupilot-os/platform/database"
	"github.com/elcomware/edupilot-os/platform/workspace"
)

var moduleKey = regexp.MustCompile("^[a-z][a-z0-9_]{0,39}$")

type Registry struct {
	Definitions []modulekit.Definition
	index       map[string]modulekit.Definition
	entries     map[string]workspace.Service
}

func NewRegistry(db *database.Catalog, definitions []modulekit.Definition) (*Registry, error) {
	registry := &Registry{Definitions: append([]modulekit.Definition(nil), definitions...), index: map[string]modulekit.Definition{}, entries: map[string]workspace.Service{}}
	for _, d := range definitions {
		if !moduleKey.MatchString(d.Key) || d.Group == "" || d.Title.EN == "" || d.Title.FR == "" {
			return nil, fmt.Errorf("invalid module definition: %q", d.Key)
		}
		if _, ok := registry.index[d.Key]; ok {
			return nil, fmt.Errorf("duplicate module %s", d.Key)
		}
		if d.Key != "students" {
			if d.Bind == nil || d.Migration == nil {
				return nil, fmt.Errorf("incomplete module %s", d.Key)
			}
			registry.entries[d.Key] = d.Bind(db)
		}
		registry.index[d.Key] = d
	}
	return registry, nil
}
func (r *Registry) Find(key string) (modulekit.Definition, bool) { d, ok := r.index[key]; return d, ok }
func (r *Registry) Entries(key string) (workspace.Service, bool) {
	s, ok := r.entries[key]
	return s, ok
}

// Enabled filters deployable modules by operator configuration. The default
// opens the complete catalog; unexpected names fail closed at startup.
func Enabled(all []modulekit.Definition, csv string) ([]modulekit.Definition, error) {
	if strings.TrimSpace(csv) == "" {
		return all, nil
	}
	wanted := map[string]bool{}
	for _, part := range strings.Split(csv, ",") {
		key := strings.TrimSpace(part)
		if key == "" || wanted[key] {
			return nil, fmt.Errorf("invalid or duplicate enabled module %q", key)
		}
		wanted[key] = true
	}
	if wanted["enrollment"] && !wanted["students"] {
		return nil, fmt.Errorf("enrollment requires students")
	}
	var selected []modulekit.Definition
	for _, d := range all {
		if wanted[d.Key] {
			selected = append(selected, d)
			delete(wanted, d.Key)
		}
	}
	if len(wanted) > 0 {
		return nil, fmt.Errorf("unknown enabled module in %q", csv)
	}
	return selected, nil
}
