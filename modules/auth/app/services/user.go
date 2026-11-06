package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/platform/access"
)

// CurrentUser reads the session, account and current role from the same school
// in one query. No caller-supplied user ID or school chooses the account.
func (s *Service) CurrentUser(ctx context.Context, token string) (models.User, error) {
	hash, err := digest(token)
	if err != nil {
		return models.User{}, err
	}
	const query = `SELECT a.id, a.email, m.school_id, m.role FROM identity_sessions s
 JOIN identity_memberships m ON m.school_id=s.school_id AND m.user_id=s.user_id
 JOIN identity_accounts a ON a.school_id=m.school_id AND a.id=m.user_id
 WHERE s.token_hash=? AND s.expires_at>?`
	var user models.User
	err = s.pool.DB.QueryRowContext(ctx, s.pool.Bind(query), hash, s.now().Unix()).Scan(&user.ID, &user.Email, &user.Membership.School, &user.Membership.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return models.User{}, access.ErrUnauthenticated
	}
	if err != nil {
		return models.User{}, err
	}
	user.Membership.UserID = user.ID
	principal, err := user.Membership.Principal()
	if err != nil {
		return models.User{}, err
	}
	user.Permissions = principal.Permissions
	return user, nil
}
