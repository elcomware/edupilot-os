# Live browser and isolation checks

Start a **disposable** native stack with `make up`. In a second terminal run:

```sh
make e2e-install
bash scripts/ci-e2e.sh
```

The script provisions random administrators in separate schools, waits for the web proxy, and runs Playwright. To select a spec, pass its path to the script, e.g. `bash scripts/ci-e2e.sh specs/isolation.spec.ts`. Archived students, file uploads and school accounts remain in this test database. Never point the script at real schools. For optional Docker use `make docker-up`, then `EDUPILOT_E2E_MODE=docker bash scripts/ci-e2e.sh`; add `DB=postgres` to both commands for the PostgreSQL overlay.

The suite exercises the real HTTP API and React UI: school isolation and forged headers, enrollment retries, private document round trips, student lifecycle, and per-user/school workspace preferences. Each browser workflow uses its own account so reloads and sign-ins do not exhaust the production login limit. Requests and credentials are not traced, recorded or captured in screenshots. Native verification runs on pushes and pull requests; the container job is optional.

To supply disposable accounts yourself, set `E2E_SCHOOL_<slot>`, `E2E_EMAIL_<slot>`, and `E2E_PASSWORD_<slot>` for each slot used by the selected tests. Use different schools for A/B (API isolation), ROSTER (student UI), PREFS/PREFS_OTHER (appearance), NAV/NAV_OTHER (navigation). Then run `make e2e`. The native base URL defaults to `http://127.0.0.1:5173`; set `E2E_BASE_URL` for another proxy. Remote test deployments require `E2E_ALLOW_REMOTE=1`.

On Linux CI use `npm exec --workspaces=false -- playwright install --with-deps chromium` from `tests/e2e`. If Chromium is already installed, set `E2E_CHROMIUM_EXECUTABLE` to its absolute path.
