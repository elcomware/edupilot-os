package controllers

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"

	"github.com/elcomware/edupilot/modules/auth/app/http/middleware"
	"github.com/elcomware/edupilot/modules/auth/app/http/responses"
	"github.com/elcomware/edupilot/modules/auth/app/services"
	authv1 "github.com/elcomware/edupilot/modules/auth/proto/v1"
	"github.com/elcomware/edupilot/platform/presentation"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// AuthController presents the shared Auth service through HTTP and gRPC.
// Lazy service resolution allows Goravel and the public web shell to start
// before a school database is provisioned.
type AuthController struct {
	authv1.UnimplementedAuthServer
	sessions func() (*services.Service, error)
	web      fs.FS
	limits   *middleware.LoginLimits
}

func New(sessions func() (*services.Service, error), web fs.FS) *AuthController {
	return &AuthController{sessions: sessions, web: web, limits: middleware.NewLoginLimits()}
}

type loginInput struct {
	School   string `json:"school"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (c *AuthController) Session(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	switch r.Method {
	case http.MethodGet:
		presentation.PageOrJSON(w, r, c.web, c.showSession)
	case http.MethodPost:
		c.login(w, r)
	case http.MethodDelete:
		c.logout(w, r)
	default:
		w.Header().Set("Allow", "POST, GET, DELETE")
		responses.Error(w, 405, "method_not_allowed", "Method not allowed.")
	}
}

func (c *AuthController) service(w http.ResponseWriter) *services.Service {
	s, err := c.sessions()
	if err != nil {
		responses.Error(w, 503, "unavailable", "School storage is unavailable.")
		return nil
	}
	return s
}

func (c *AuthController) showSession(w http.ResponseWriter, r *http.Request) {
	s := c.service(w)
	if s == nil {
		return
	}
	p, err := middleware.RequestPrincipal(s, r)
	if err != nil {
		responses.ServiceError(w, err)
		return
	}
	presentation.JSON(w, http.StatusOK, p)
}

func (c *AuthController) login(w http.ResponseWriter, r *http.Request) {
	if !c.limits.AllowSource(r.RemoteAddr) {
		throttled(w)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8192)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var in loginInput
	if err := dec.Decode(&in); err != nil || dec.Decode(new(any)) != io.EOF {
		responses.Error(w, 400, "invalid_request", "Expected a JSON object with school, email and password.")
		return
	}
	release, ok := c.limits.AcquireAccount(in.School, in.Email)
	if !ok {
		throttled(w)
		return
	}
	defer release()
	s := c.service(w)
	if s == nil {
		return
	}
	session, err := s.Login(r.Context(), in.School, in.Email, in.Password)
	if err != nil {
		responses.ServiceError(w, err)
		return
	}
	presentation.JSON(w, http.StatusOK, session)
}

func (c *AuthController) logout(w http.ResponseWriter, r *http.Request) {
	s := c.service(w)
	if s == nil {
		return
	}
	if _, err := middleware.RequestPrincipal(s, r); err != nil {
		responses.ServiceError(w, err)
		return
	}
	token, err := middleware.Bearer(r.Header.Values("Authorization"))
	if err == nil {
		err = s.Logout(r.Context(), token)
	}
	if err != nil {
		responses.ServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func throttled(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "60")
	responses.Error(w, 429, "rate_limited", "Too many sign-in attempts. Try again later.")
}

func (c *AuthController) CurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	s := c.service(w)
	if s == nil {
		return
	}
	user, err := middleware.SessionUser(r.Context(), s, r.Header.Values("Authorization"), r.Header.Values("X-School-ID"))
	if err != nil {
		responses.ServiceError(w, err)
		return
	}
	presentation.JSON(w, http.StatusOK, user)
}

func (c *AuthController) GetCurrentUser(ctx context.Context, _ *authv1.GetCurrentUserRequest) (*authv1.CurrentUser, error) {
	s, err := c.sessions()
	if err != nil {
		return nil, status.Error(codes.Unavailable, "School storage is unavailable.")
	}
	md, _ := metadata.FromIncomingContext(ctx)
	user, err := middleware.SessionUser(ctx, s, md.Get("authorization"), md.Get("x-school-id"))
	if err != nil {
		return nil, responses.RPCError(err)
	}
	return &authv1.CurrentUser{Id: user.ID, Email: user.Email, Membership: &authv1.Membership{School: user.Membership.School, UserId: user.Membership.UserID, Role: string(user.Membership.Role)}, Permissions: user.Permissions}, nil
}
