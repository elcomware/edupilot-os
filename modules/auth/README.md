# Auth

Auth is the reference module for EduPilot OS. The host starts Goravel once. Auth owns sign-in, sign-out, school memberships and the permission policy for the three starter roles.

## Follow a sign-in

1. The web and Wails clients render `resources/js/pages/Login.tsx`. Expo renders `resources/js/native/Login.tsx`. Both use the host client and session store.
2. `routes/web.go`, `routes/api.go` and `routes/grpc.go` register the three transports in Goravel. One `AuthController` presents them. The host resolves the control connection only when a request needs it.
3. `app/http/controllers/auth_controller.go` reads credentials, applies the limits in `app/http/middleware/login_limits.go` and invokes `app/services/session.go`. Credentials and sessions live on the control connection even with a dedicated school database.
4. The service joins the account to the membership on **both school ID and user ID**. `app/models/membership.go` links the user to a role. `app/models/role.go` defines that role’s effective permissions. Changing a member's role affects their next request with the same token.
5. `GET /api/v1/auth/me` returns the caller's ID, email, school membership, role and effective permissions. The query in `app/services/user.go` reads all of them from one session-scoped snapshot. An asserted school must match. The existing session endpoint retains its original response contract.

## Routes and response formats

| Request | Result |
| --- | --- |
| `GET /login` in a browser | The shared platform presenter serves the compiled React shell. Its Auth sign-in page submits to the versioned session endpoint and opens the school workspace after sign-in. |
| `GET /login` with `Accept: application/json` | The same controller reads the authenticated session and returns its principal. A missing or revoked bearer returns 401. Web responses vary on `Accept`, honor media weights and return 406 for unsupported formats. |
| `GET /api/v1/session`, `POST /api/v1/session`, `DELETE /api/v1/session` | Read the principal, sign in or revoke the session. Versioned routes keep their JSON contracts. A successful sign-out returns 204. |
| `GET /api/v1/auth/me` | An authenticated account, its school membership and its current permissions as JSON. |
| `edupilot.auth.v1.Auth/GetCurrentUser` | The same scoped account as a typed protobuf response. Send one `authorization: Bearer <session>` metadata value and optionally one `x-school-id`. Invalid credentials return `Unauthenticated`. A conflicting school returns `PermissionDenied`. |

HTTP representation selection lives in [the platform presenter](../../platform/presentation/http.go). The browser uses the existing React build and keeps its bearer in memory. HTML contains only the public application shell. Auth services own membership lookup, permissions and session changes.

Use `make up` and open `http://127.0.0.1:3000/login` or `http://127.0.0.1:5173/login`. For a standalone Go server, first run `npm run build:web` and keep `clients/web/dist/` under the application root. The optional Compose web gateway serves the same route.

The native gRPC listener is opt-in. For a local drill, set `GRPC_HOST=127.0.0.1` and `GRPC_PORT=50051` before starting the server. To listen on a network interface, set `GRPC_TLS_CERT` and `GRPC_TLS_KEY` to the server certificate and private key paths. Clients must verify the certificate. The versioned client and server contract is [auth.proto](proto/v1/auth.proto). Regenerate the checked-in Go types from the repository root with `protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. modules/auth/proto/v1/auth.proto` using protoc-gen-go v1.36.12 and protoc-gen-go-grpc v1.5.1.

| Relationship | Current implementation |
| --- | --- |
| Account to school membership | One row per school and user, constrained by a composite foreign key. Identical email addresses at two schools remain separate. |
| Membership to role | One of admin, registrar or viewer per legacy school membership. |
| Role to permissions | Fixed policy in the Auth model. Registrar can change student records. Viewer can read them. Only the school administrator can manage memberships. |
| Membership to sessions | Multiple independently revocable sessions. Only token digests are stored. |

`database/migrations/` holds the historical identity schema and the additive group-tenancy schema. `providers/` registers the module's database definitions with Goravel. To explicitly provision a school administrator, supply the same environment settings as the existing `seed-admin` command and run `./artisan db:seed --seeder=AuthSchoolAdminSeeder`. Provisioning checks the migration ledger. Repeating it with the original credentials verifies the existing school administrator.

The public module key remains `identity` for compatibility with existing workspace URLs, preferences, migrations and stored rows. The Users and Permissions catalog work lists remain separate from authoritative account and grant records. Tenant-wide roles, custom grants and multiple roles per scope belong to the group-tenancy cutover. A school administrator has no platform-operator rights.

## Module convention

Use Goravel names such as `app/models`, `app/services`, `app/contracts`, `app/http/controllers`, `database/migrations`, `database/seeders`, `providers`, `routes` and `resources` for the parts a module implements. Keep shared UI controls in the client shell. Register routes for real transports only. Shared infrastructure stays in the host. Other business modules use public contracts or the platform's verified principal.

The convention follows the [Gorazana Modules generator](https://github.com/gorazana/modules/tree/0fa6d626c042f65e7494c9b4cea2378fc4830dda). Its candidate supports module-local resources, split providers and Inertia React, and targets Go 1.26.10 and Goravel 1.18. The candidate is private and untagged at review. EduPilot builds use the checked-in host registration and generator. Qualify the package in a disposable host against EduPilot's pinned framework, role checks, frozen migration history, PostgreSQL and client builds before adding the runtime dependency.
