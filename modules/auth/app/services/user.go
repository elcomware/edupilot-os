package services

import (
	"context"
	"database/sql"

	"github.com/elcomware/edupilot/modules/auth/app/models"
	"github.com/elcomware/edupilot/modules/core/access"
)

func (s *Service) CurrentUser(ctx context.Context, token string) (models.User, error) {
	return s.currentUser(ctx, s.pool.DB, token)
}

// Read the session and all current grants from one school-scoped snapshot.
func (s *Service) currentUser(ctx context.Context, q queryer, token string) (models.User, error) {
	hash, err := digest(token)
	if err != nil {
		return models.User{}, err
	}
	const statement = `SELECT u.id,u.email,m.school_id,m.role,s.expires_at,g.permission
 FROM auth_sessions s JOIN auth_memberships m ON m.school_id=s.school_id AND m.user_id=s.user_id
 JOIN auth_users u ON u.school_id=m.school_id AND u.id=m.user_id
 LEFT JOIN auth_role_permissions g ON g.school_id=m.school_id AND g.role=m.role
 WHERE s.token_hash=? AND s.expires_at>? ORDER BY g.permission`
	rows, err := q.QueryContext(ctx, s.pool.Bind(statement), hash, s.now().Unix())
	if err != nil {
		return models.User{}, err
	}
	defer rows.Close()
	user := models.User{Permissions: []string{}}
	for rows.Next() {
		var grant sql.NullString
		if err := rows.Scan(&user.ID, &user.Email, &user.Membership.School, &user.Membership.Role, &user.ExpiresAt, &grant); err != nil {
			return models.User{}, err
		}
		if grant.Valid {
			user.Permissions = append(user.Permissions, grant.String)
		}
	}
	if err := rows.Err(); err != nil {
		return models.User{}, err
	}
	if user.ID == "" {
		return models.User{}, access.ErrUnauthenticated
	}
	user.Membership.UserID = user.ID
	return user, nil
}
