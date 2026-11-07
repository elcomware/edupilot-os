package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"math"

	"github.com/elcomware/edupilot/modules/core/access"
	"github.com/elcomware/edupilot/modules/core/database"
	"github.com/elcomware/edupilot/modules/core/outbox"
	"github.com/elcomware/edupilot/modules/core/scope"
)

type TenantGrantInput struct {
	ID         string         `json:"id"`
	Subject    access.Subject `json:"subject"`
	Tenant     scope.TenantID `json:"tenant"`
	Permission string         `json:"permission"`
	ExpiresAt  int64          `json:"expiresAt"`
}

type TenantGrant struct {
	TenantGrantInput
	State     string `json:"state"`
	Version   uint64 `json:"version"`
	CreatedAt int64  `json:"createdAt"`
	RevokedAt int64  `json:"revokedAt,omitempty"`
}

func (in TenantGrantInput) Validate() error {
	_, err := scope.Parse[scope.Membership](in.ID)
	if err != nil || !database.ValidTenant(in.Subject.Realm) || len(in.Subject.ID) < 1 || len(in.Subject.ID) > 32 || in.Tenant.IsZero() || in.Permission != access.HierarchyRead || in.ExpiresAt <= 0 {
		return ErrInvalidInput
	}
	return nil
}

// LockTenantGrantSubject precedes the target gate lock and excludes membership
// deletion until the control grant commit. A retry also requires membership.
func (s *Service) LockTenantGrantSubject(ctx context.Context, tx *sql.Tx, in TenantGrantInput) error {
	if err := in.Validate(); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, s.pool.Bind("UPDATE auth_memberships SET role=role WHERE school_id=? AND user_id=?"), in.Subject.Realm, in.Subject.ID)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return ErrNotFound
	}
	return nil
}

const grantColumns = "id,realm_id,user_id,tenant_id,permission,expires_at,state,version,created_at,revoked_at"

type grantScanner interface{ Scan(...any) error }

func scanTenantGrant(row grantScanner) (TenantGrant, error) {
	var g TenantGrant
	var tenant string
	var revoked sql.NullInt64
	err := row.Scan(&g.ID, &g.Subject.Realm, &g.Subject.ID, &tenant, &g.Permission, &g.ExpiresAt, &g.State, &g.Version, &g.CreatedAt, &revoked)
	if err != nil {
		return g, err
	}
	g.Tenant, err = scope.Parse[scope.Tenant](tenant)
	g.RevokedAt = revoked.Int64
	if err != nil || g.Validate() != nil || g.Version < 1 || g.Version > math.MaxInt64 || g.CreatedAt <= 0 || (g.State != "active" && g.State != "revoked") || (g.State == "active" && revoked.Valid) || (g.State == "revoked" && !revoked.Valid) {
		return TenantGrant{}, ErrConflict
	}
	return g, nil
}

// CreateTenantGrant runs inside the coordinator's placement/Auth/target fences.
// Only control is written. Expiration and revocation are terminal in this release.
func (s *Service) CreateTenantGrant(ctx context.Context, tx *sql.Tx, in TenantGrantInput) (TenantGrant, error) {
	if err := in.Validate(); err != nil {
		return TenantGrant{}, err
	}
	now := s.now().Unix()
	if in.ExpiresAt <= now {
		return TenantGrant{}, ErrConflict
	}
	g := TenantGrant{TenantGrantInput: in, State: "active", Version: 1, CreatedAt: now}
	result, err := tx.ExecContext(ctx, s.pool.Bind(`INSERT INTO auth_tenant_grants
 (id,realm_id,user_id,tenant_id,permission,expires_at,state,version,created_at)
 VALUES (?,?,?,?,?,?,'active',1,?) ON CONFLICT DO NOTHING`), in.ID, in.Subject.Realm, in.Subject.ID, in.Tenant.String(), in.Permission, in.ExpiresAt, now)
	if err != nil {
		return TenantGrant{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return TenantGrant{}, err
	}
	if n == 0 {
		saved, err := scanTenantGrant(tx.QueryRowContext(ctx, s.pool.Bind("SELECT "+grantColumns+" FROM auth_tenant_grants WHERE id=?"), in.ID))
		if errors.Is(err, sql.ErrNoRows) {
			return TenantGrant{}, ErrConflict
		}
		if err != nil {
			return TenantGrant{}, err
		}
		if saved.TenantGrantInput != in || saved.State != "active" || saved.Version != 1 || saved.ExpiresAt <= now {
			return TenantGrant{}, ErrConflict
		}
		return saved, nil
	}
	if err := s.grantEvent(ctx, tx, g, "auth.tenant-grant.created.v1"); err != nil {
		return TenantGrant{}, err
	}
	return g, nil
}

func (s *Service) grantEvent(ctx context.Context, tx *sql.Tx, g TenantGrant, kind string) error {
	payload, err := json.Marshal(g)
	if err != nil {
		return err
	}
	return outbox.Append(ctx, tx, *s.pool, outbox.Event{TenantID: g.Tenant.String(), AggregateID: g.ID, Type: kind, Payload: payload})
}

// RevokeTenantGrant needs only control storage, including when target is offline.
func (s *Service) RevokeTenantGrant(ctx context.Context, id string, version uint64) (TenantGrant, error) {
	if _, err := scope.Parse[scope.Membership](id); err != nil || version == 0 || version >= math.MaxInt64 {
		return TenantGrant{}, ErrInvalidInput
	}
	tx, err := s.pool.DB.BeginTx(ctx, nil)
	if err != nil {
		return TenantGrant{}, err
	}
	defer tx.Rollback()
	// Acquire a row/write lock before reading, so competing retries see completion.
	if _, err := tx.ExecContext(ctx, s.pool.Bind("UPDATE auth_tenant_grants SET version=version WHERE id=?"), id); err != nil {
		return TenantGrant{}, err
	}
	g, err := scanTenantGrant(tx.QueryRowContext(ctx, s.pool.Bind("SELECT "+grantColumns+" FROM auth_tenant_grants WHERE id=?"), id))
	if errors.Is(err, sql.ErrNoRows) {
		return TenantGrant{}, ErrNotFound
	}
	if err != nil {
		return TenantGrant{}, err
	}
	if g.State == "revoked" && g.Version == version+1 {
		return g, nil
	}
	if g.State != "active" || g.Version != version {
		return TenantGrant{}, ErrConflict
	}
	g.State, g.Version, g.RevokedAt = "revoked", version+1, s.now().Unix()
	if _, err := tx.ExecContext(ctx, s.pool.Bind("UPDATE auth_tenant_grants SET state='revoked',version=?,revoked_at=? WHERE id=?"), g.Version, g.RevokedAt, id); err != nil {
		return TenantGrant{}, err
	}
	if err := s.grantEvent(ctx, tx, g, "auth.tenant-grant.revoked.v1"); err != nil {
		return TenantGrant{}, err
	}
	if err := tx.Commit(); err != nil {
		return TenantGrant{}, err
	}
	return g, nil
}
