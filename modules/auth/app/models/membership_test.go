package models

import (
	"errors"
	"slices"
	"testing"

	"github.com/elcomware/edupilot/modules/core/access"
)

func TestMembershipPermissions(t *testing.T) {
	for _, tc := range []struct {
		role          string
		write, manage bool
	}{{RoleViewer, false, false}, {RoleRegistrar, true, false}, {RoleAdmin, true, true}} {
		t.Run(tc.role, func(t *testing.T) {
			grants, err := Role(tc.role).DefaultPermissions()
			p := access.Principal{ID: "member", School: "north", Permissions: grants}
			if err != nil || !p.Allows(access.WorkspaceRead) || p.Allows(access.WorkspaceWrite) != tc.write || p.Allows(access.ManageMembers) != tc.manage {
				t.Fatalf("incorrect %s grants: %v, %v", tc.role, p, err)
			}
			if err = p.Require("south", access.WorkspaceRead); !errors.Is(err, access.ErrForbidden) {
				t.Fatal("cross-school authority")
			}
			if slices.Contains(p.Permissions, "*") {
				t.Fatal("unexpected wildcard permission")
			}
		})
	}
	if _, err := Role("unknown").DefaultPermissions(); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("unknown role accepted")
	}
}
