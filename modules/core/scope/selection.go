package scope

import "errors"

var ErrUnsupported = errors.New("scope: unsupported selection")
var ErrUnknown = errors.New("scope: unknown or inconsistent hierarchy")

// Selection asks for a working context. Provided ancestors must agree with the
// selected descendant. Team validation requires separate membership and grants.
type Selection struct {
	Tenant       TenantID       `json:"tenant"`
	Organization OrganizationID `json:"organization,omitzero"`
	School       SchoolID       `json:"school,omitzero"`
	Campus       CampusID       `json:"campus,omitzero"`
	Team         TeamID         `json:"team,omitzero"`
}

// Chain records verified topology. The caller must also verify the subject's
// effective grants before reading or writing business records.
type Chain struct {
	Tenant       TenantID       `json:"tenant"`
	Organization OrganizationID `json:"organization,omitzero"`
	School       SchoolID       `json:"school,omitzero"`
	Campus       CampusID       `json:"campus,omitzero"`
	// The root comes first, the selected or owning organization last.
	Ancestors        []OrganizationID `json:"ancestors,omitempty"`
	HierarchyVersion uint64           `json:"hierarchyVersion"`
	Active           bool             `json:"active"`
}
