package controllers

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	authv1 "github.com/elcomware/edupilot/modules/auth/proto/v1"
	"github.com/elcomware/edupilot/modules/core/access"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestHTTPAndGRPCShareMembershipAndRevocation(t *testing.T) {
	s := fixture(t)
	c := testController(s)
	ctx := t.Context()
	const pass = "disposable-auth-fixture"
	if _, err := s.Seed(ctx, "north", "admin@example.test", pass); err != nil {
		t.Fatal(err)
	}
	admin, err := s.Login(ctx, "north", "admin@example.test", pass)
	if err != nil {
		t.Fatal(err)
	}
	member, err := s.CreateMember(ctx, admin.Token, "north", "member@example.test", pass, models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	session, err := s.Login(ctx, "north", "member@example.test", pass)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	authv1.RegisterAuthServer(server, c)
	go server.Serve(listener)
	t.Cleanup(server.Stop)
	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	client := authv1.NewAuthClient(conn)
	bearer := "Bearer " + session.Token
	rpc := func(want codes.Code, pairs ...string) *authv1.CurrentUser {
		t.Helper()
		result, err := client.GetCurrentUser(metadata.NewOutgoingContext(ctx, metadata.Pairs(pairs...)), &authv1.GetCurrentUserRequest{})
		if status.Code(err) != want {
			t.Fatalf("RPC status = %v, want %v", status.Code(err), want)
		}
		return result
	}
	rpc(codes.Unauthenticated)
	rpc(codes.Unauthenticated, "authorization", bearer, "authorization", bearer)
	rpc(codes.PermissionDenied, "authorization", bearer, "x-school-id", "south")
	rpc(codes.PermissionDenied, "authorization", bearer, "x-school-id", "north", "x-school-id", "north")
	for _, role := range []string{models.RoleViewer, models.RoleRegistrar} {
		if role == models.RoleRegistrar {
			if err := s.SetMemberRole(ctx, admin.Token, "north", member.ID, role); err != nil {
				t.Fatal(err)
			}
		}
		profile := rpc(codes.OK, "authorization", bearer, "x-school-id", "north")
		r := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
		r.Header.Set("Authorization", bearer)
		w := httptest.NewRecorder()
		c.CurrentUser(w, r)
		var web models.User
		if w.Code != http.StatusOK || json.Unmarshal(w.Body.Bytes(), &web) != nil || profile.Id != web.ID || profile.Email != web.Email || profile.Membership.School != web.Membership.School || profile.Membership.UserId != web.Membership.UserID || profile.Membership.Role != role || string(web.Membership.Role) != role || !slices.Equal(profile.Permissions, web.Permissions) || slices.Contains(profile.Permissions, access.WorkspaceWrite) != (role == models.RoleRegistrar) {
			t.Fatal("HTTP and RPC returned an incorrect membership")
		}
	}
	if err := s.Logout(ctx, session.Token); err != nil {
		t.Fatal(err)
	}
	rpc(codes.Unauthenticated, "authorization", bearer)
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := client.GetCurrentUser(cancelled, &authv1.GetCurrentUserRequest{}); status.Code(err) != codes.Canceled {
		t.Fatal("cancellation lost")
	}
}
