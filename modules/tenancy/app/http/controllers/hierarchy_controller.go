package controllers

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/presentation"
	"github.com/elcomware/edupilot/modules/core/scope"
	"github.com/elcomware/edupilot/modules/tenancy/app/models"
	"github.com/elcomware/edupilot/modules/tenancy/app/services"
)

type HierarchyController struct {
	Authorizer func() (access.TenantAuthorizer, error)
	Reader     func() (services.HierarchyReader, error)
}

func (c HierarchyController) List(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	q, limit, err := access.ScopeQuery(r, "tenant", "kind", "organization", "school", "limit", "cursor")
	if err != nil {
		access.ScopeError(w, err)
		return
	}
	token, err := access.Bearer(r.Header.Values("Authorization"))
	if err != nil {
		access.ScopeError(w, err)
		return
	}
	tenant, err := scope.Parse[scope.Tenant](q.Get("tenant"))
	if err != nil {
		access.ScopeError(w, access.ErrInvalidScope)
		return
	}
	authorizer, err := c.Authorizer()
	if err != nil {
		access.ScopeError(w, err)
		return
	}
	verified, err := authorizer.AuthorizeTenant(ctx, token, r.Header.Values("X-School-ID"), tenant)
	if err != nil {
		access.ScopeError(w, err)
		return
	}
	query := services.HierarchyQuery{Tenant: tenant, Kind: q.Get("kind"), Limit: limit, Cursor: q.Get("cursor")}
	if q.Has("organization") {
		query.Organization, err = scope.Parse[scope.Organization](q.Get("organization"))
		if err != nil {
			access.ScopeError(w, access.ErrInvalidScope)
			return
		}
	}
	if q.Has("school") {
		query.School, err = scope.Parse[scope.School](q.Get("school"))
		if err != nil {
			access.ScopeError(w, access.ErrInvalidScope)
			return
		}
	}
	reader, err := c.Reader()
	if err != nil {
		access.ScopeError(w, err)
		return
	}
	page, err := reader.ListAuthorized(ctx, verified, query)
	switch {
	case err == nil:
		presentation.JSON(w, http.StatusOK, page)
	case errors.Is(err, models.ErrInvalid):
		access.ScopeError(w, access.ErrInvalidScope)
	case errors.Is(err, models.ErrConflict):
		access.ScopeHTTPError(w, 409, "stale_directory", "Directory changed. Start a new listing.")
	case errors.Is(err, database.ErrPlacement):
		access.ScopeError(w, err)
	case errors.Is(err, models.ErrScope):
		access.ScopeHTTPError(w, 404, "not_found", "Hierarchy owner not found.")
	default:
		access.ScopeError(w, err)
	}
}
