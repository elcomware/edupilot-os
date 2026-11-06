# Auth

Auth owns users, credentials, roles, permissions, school memberships and revocable sessions. Start with `routes/web.go`, `routes/api.go` or `routes/grpc.go`. They share one controller and application services.

| Path | Responsibility |
|---|---|
| `app/services/guard.go` | Goravel’s default school guard. A consumer calls `facades.Auth(ctx).User(&principal)` using `modules/core/access.Principal`. |
| `providers/auth_service_provider.go` | Registers the session service in Goravel IoC and attaches the guard using `Auth.Extend`. |
| `database/migrations/schema.go` | Goravel Blueprint definitions for school realms, users, roles, permissions, role grants, memberships and sessions on the control pool. |
| `database/seeders/` | Explicit school-administrator provisioning through Artisan. Default school grants are inserted once with the first administrator. |
| `app/services/user.go` | Reads the current session, school membership and grants in one query. Revocation and grant changes apply on the next request. |
| `resources/js/` | Sign-in and My account pages for web/desktop and native mobile. |

A membership references one role in its own school. Each role has a set of grants through `auth_role_permissions`. The default roles are administrator, registrar and viewer. A new sign-in requires credentials.

Other business modules use the request-scoped guard, a Core contract injected at bootstrap, or a versioned event.
