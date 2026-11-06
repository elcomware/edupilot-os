package models

import (
	"errors"
	"github.com/elcomware/edupilot/platform/access"
	"slices"
	"testing"
)

func TestMembershipPermissions(t *testing.T) {
	for _, tc := range []struct {
		role          string
		write, manage bool
	}{{RoleViewer, false, false}, {RoleRegistrar, true, false}, {RoleAdmin, true, true}} {
		t.Run(tc.role, func(t *testing.T) {
			p, err := (Membership{UserID: "member", School: "north", Role: Role(tc.role)}).Principal()
			if err != nil || !p.Allows(access.StudentsRead) || p.Allows(access.StudentsWrite) != tc.write || p.Allows(access.ManageMembers) != tc.manage {
				t.Fatalf("incorrect %s grants: %v, %v", tc.role, p, err)
			}
			if err = p.Require("south", access.StudentsRead); !errors.Is(err, access.ErrForbidden) {
				t.Fatal("cross-school authority")
			}
			if slices.Contains(p.Permissions, "*") {
				t.Fatal("unexpected wildcard permission")
			}
		})
	}
	if _, err := (Membership{UserID: "id", School: "north", Role: "unknown"}).Principal(); !errors.Is(err, access.ErrForbidden) {
		t.Fatal("unknown role accepted")
	}
}
