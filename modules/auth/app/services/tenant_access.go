package services

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"

	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/scope"
)

func (s *Service) tenantSnapshot(ctx context.Context, token string, realms []string) (*sql.Tx, access.Subject, error) {
	opts := &sql.TxOptions{ReadOnly: true}
	if s.pool.Driver == "postgres" {
		opts.Isolation = sql.LevelRepeatableRead
	}
	tx, err := s.pool.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, access.Subject{}, err
	}
	user, err := s.currentUser(ctx, tx, token)
	if err == nil && len(realms) > 0 && (len(realms) != 1 || realms[0] != user.Membership.School) {
		err = access.ErrForbidden
	}
	if err != nil {
		tx.Rollback()
		return nil, access.Subject{}, err
	}
	return tx, access.Subject{Realm: user.Membership.School, ID: user.ID}, nil
}

// AuthorizeTenant reads session, live membership and explicit grant from one
// control snapshot per request. Legacy role grants never supply tenant authority.
func (s *Service) AuthorizeTenant(ctx context.Context, token string, realms []string, tenant scope.TenantID) (access.TenantAuthorization, error) {
	if tenant.IsZero() {
		return access.TenantAuthorization{}, access.ErrInvalidScope
	}
	tx, subject, err := s.tenantSnapshot(ctx, token, realms)
	if err != nil {
		return access.TenantAuthorization{}, err
	}
	defer tx.Rollback()
	g, err := scanTenantGrant(tx.QueryRowContext(ctx, s.pool.Bind("SELECT "+grantColumns+" FROM auth_tenant_grants WHERE realm_id=? AND user_id=? AND tenant_id=? AND permission=? AND state='active' AND expires_at>?"), subject.Realm, subject.ID, tenant.String(), access.HierarchyRead, s.now().Unix()))
	if errors.Is(err, sql.ErrNoRows) {
		return access.TenantAuthorization{}, access.ErrForbidden
	}
	if err != nil {
		return access.TenantAuthorization{}, err
	}
	if err := tx.Commit(); err != nil {
		return access.TenantAuthorization{}, err
	}
	return tenantAuthorization(g), nil
}

func tenantAuthorization(g TenantGrant) access.TenantAuthorization {
	return access.TenantAuthorization{Subject: g.Subject, Tenant: g.Tenant, Permission: g.Permission, GrantID: g.ID, Version: g.Version, ExpiresAt: g.ExpiresAt}
}

type TenantContexts struct {
	Entries    []access.TenantAuthorization `json:"entries"`
	NextCursor string                       `json:"nextCursor,omitempty"`
}

type contextCursor struct {
	Format  int            `json:"format"`
	Subject access.Subject `json:"subject"`
	After   string         `json:"after"`
}

// TenantContexts supplies live keyset pages. Its unsigned cursor conveys no
// authority, and entries added behind the cursor require a fresh enumeration.
func (s *Service) TenantContexts(ctx context.Context, token string, realms []string, limit int, cursor string) (TenantContexts, error) {
	if limit < 1 || limit > 100 || len(cursor) > 1024 {
		return TenantContexts{}, access.ErrInvalidScope
	}
	tx, subject, err := s.tenantSnapshot(ctx, token, realms)
	if err != nil {
		return TenantContexts{}, err
	}
	defer tx.Rollback()
	continuation := contextCursor{Format: 1, Subject: subject}
	if cursor != "" {
		data, err := base64.RawURLEncoding.Strict().DecodeString(cursor)
		if err != nil {
			return TenantContexts{}, access.ErrInvalidScope
		}
		var saved contextCursor
		if json.Unmarshal(data, &saved) != nil {
			return TenantContexts{}, access.ErrInvalidScope
		}
		canonical, _ := json.Marshal(saved)
		if !bytes.Equal(data, canonical) || saved.Format != 1 || saved.Subject != subject {
			return TenantContexts{}, access.ErrInvalidScope
		}
		if _, err := scope.Parse[scope.Tenant](saved.After); err != nil {
			return TenantContexts{}, access.ErrInvalidScope
		}
		continuation = saved
	}
	rows, err := tx.QueryContext(ctx, s.pool.Bind("SELECT "+grantColumns+" FROM auth_tenant_grants WHERE realm_id=? AND user_id=? AND permission=? AND state='active' AND expires_at>? AND tenant_id>? ORDER BY tenant_id LIMIT ?"), subject.Realm, subject.ID, access.HierarchyRead, s.now().Unix(), continuation.After, limit+1)
	if err != nil {
		return TenantContexts{}, err
	}
	defer rows.Close()
	page := TenantContexts{Entries: make([]access.TenantAuthorization, 0, limit)}
	for rows.Next() {
		g, err := scanTenantGrant(rows)
		if err != nil {
			return TenantContexts{}, err
		}
		if len(page.Entries) == limit {
			continuation.After = page.Entries[len(page.Entries)-1].Tenant.String()
			data, _ := json.Marshal(continuation)
			page.NextCursor = base64.RawURLEncoding.EncodeToString(data)
			break
		}
		page.Entries = append(page.Entries, tenantAuthorization(g))
	}
	if err := rows.Err(); err != nil {
		return TenantContexts{}, err
	}
	if err := rows.Close(); err != nil {
		return TenantContexts{}, err
	}
	if err := tx.Commit(); err != nil {
		return TenantContexts{}, err
	}
	return page, nil
}
