# Live browser and tenant checks

The runner opens the actual React client, authenticates with the Goravel API, and writes real records. Use two **disposable school administrators**, one per school. Students remain archived for audit. The appearance check uses a fresh Playwright browser context. No mock server, network interception, default accounts, or demo rows are used.

From the repository root, start the chosen stack with `make up` (SQLite) or `make up DB=postgres`. Install the isolated runner and Chromium with `make e2e-install`. On Linux CI, the equivalent browser installation command is `cd tests/e2e && npm exec --workspaces=false -- playwright install --with-deps chromium`.

- For a disposable local stack, `bash scripts/ci-e2e.sh` creates two unique schools with random credentials and runs the suite. Use `DB=postgres bash scripts/ci-e2e.sh` for the PostgreSQL stack.
- To supply existing accounts, export `E2E_SCHOOL_A`, `E2E_EMAIL_A`, `E2E_PASSWORD_A` and the matching three variables ending in `_B`. Then run `make e2e`. Use only schools reserved for these tests.
- `E2E_BASE_URL` defaults to `http://127.0.0.1:8080`, the Compose web proxy. For Vite use `http://127.0.0.1:5173` with both the API and Vite running. Remote deployments require explicit `E2E_ALLOW_REMOTE=1`.

The suite checks sign-in, registration, edit, reload, archival, stored version and status, durable appearance settings, and read/write isolation using two independent sessions. A forged school header must be rejected. Request traces are disabled to avoid retaining credentials or bearer tokens in output.

The package is intentionally outside the app's npm workspaces. To select a spec, run `cd tests/e2e && npm run test --workspaces=false -- specs/isolation.spec.ts`. GitHub Actions only runs the verification workflow when manually dispatched.

The connected-module specifications check Enrollment retry behavior and binary Documents round trips through the web proxy. They create files in the disposable school's storage. Run them only with test schools.

The native stack uses `http://127.0.0.1:5173` (`E2E_BASE_URL`). Run
`EDUPILOT_E2E_MODE=native bash scripts/ci-e2e.sh` against a **disposable**
native stack to provision temporary schools and run the suite. The optional
container job sets `EDUPILOT_E2E_MODE=docker` and uses port 8080.
If Chromium is already installed, set `E2E_CHROMIUM_EXECUTABLE` to its absolute
executable path; otherwise install the Playwright-managed browser. This
changes only the browser executable, not assertions or application behavior.
