package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/elcomware/edupilot/modules/core/access"
	contracts "github.com/elcomware/edupilot/modules/core/contracts/tenancy"
	"github.com/elcomware/edupilot/modules/core/presentation"
	"github.com/elcomware/edupilot/modules/core/scope"
)

// Contexts validates every returned choice against live placement and gate state.
// Any unavailable/closed choice fails the whole page with 503.
func (c *AuthController) Contexts(directory func() (contracts.Directory, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		query, limit, err := access.ScopeQuery(r, "limit", "cursor")
		if err != nil {
			access.ScopeError(w, err)
			return
		}
		token, err := access.Bearer(r.Header.Values("Authorization"))
		if err != nil {
			access.ScopeError(w, err)
			return
		}
		service, err := c.sessions()
		if err != nil {
			access.ScopeError(w, err)
			return
		}
		page, err := service.TenantContexts(ctx, token, r.Header.Values("X-School-ID"), limit, query.Get("cursor"))
		if err != nil {
			access.ScopeError(w, err)
			return
		}
		d, err := directory()
		if err != nil {
			access.ScopeError(w, err)
			return
		}
		for _, entry := range page.Entries {
			chain, err := d.Resolve(ctx, scope.Selection{Tenant: entry.Tenant})
			if err != nil || !chain.Active {
				access.ScopeHTTPError(w, 503, "unavailable", "Tenant directory is unavailable.")
				return
			}
		}
		presentation.JSON(w, http.StatusOK, page)
	}
}
