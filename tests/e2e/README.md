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

Native CI saves three explicit screenshots from disposable successful sign-ins: the desktop directory, desktop preferences and the narrow French preferences page. It captures only these two routes on loopback, with the generated school identifier hidden. Find them in the `workspace-layouts` artifact for 14 days.

The existing navigation and student lifecycle journeys also run pinned `axe-core`
against the actual page, with no extra sign-ins. Checkpoints cover sign-in,
directory and preferences at desktop/light and 320px French/dark, the shared
reset dialog, and the populated student register with its column disclosure
open in both layouts. WCAG 2.0/2.1 A and AA and 2.2 AA tagged rules run without
rule exclusions. A violation fails the journey; an engine error also fails.
Incomplete results are explicitly recorded for manual review, not counted as
passing rules. Keyboard regressions check column disclosure activation and
checkbox focus, dialog focus wrap/Escape/trigger restoration, and narrow page
reflow. The original reload, scoped preferences and student lifecycle assertions
still run on the same accounts.

Each checkpoint attaches a small `accessibility-*.json` report containing only
the engine version, selected tags, passing-rule count, and violated/incomplete
rule IDs with structural child-position targets. These targets can locate a
node in the same page state without exposing IDs, attributes, page HTML, record
text or credentials. Do not attach raw axe results, DOM snapshots, or sign-in
captures. Native CI saves these reports in `accessibility-evidence` for 14 days
and audits the separate browser-runner dependency lockfile. Reproduce with
`bash scripts/ci-e2e.sh specs/workspace-navigation.spec.ts specs/students.spec.ts`.

Automated scans and keyboard regressions are partial evidence, not WCAG
certification. Incomplete findings, manual screen-reader use, focus visibility
and contrast in additional themes, 200% zoom, and actual supported devices
remain separate release checks. If the browser cannot launch or a journey
cannot reach a checkpoint, that checkpoint remains unverified; test discovery
or a client build does not count as a browser pass.
