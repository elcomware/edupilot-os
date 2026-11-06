# Verification

Run `go test -count=1 ./...`, `go vet ./...` and `go build ./...`. Use the PostgreSQL environment variables in CI to include the PostgreSQL foundation drill. The drill creates its own databases and verifies placement markers, migration retry, live grants, cache behavior, school mismatch and revocation.

Run `make test-web test-mobile`, `npm run build:web` and `npm run export --workspace @edupilot/mobile`. Exercise Wails and Expo through their native workflows. Run `bash scripts/ci-e2e.sh` only against a disposable stack. It provisions synthetic schools and checks access, isolation, sign-out, navigation and preferences.
