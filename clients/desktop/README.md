# EduPilot OS desktop

This is a native Wails v3.1.0 host for the **same compiled web UI** served by Goravel. It uses the same login, language, navigation, module pages and student forms. It has no Go business services, listener, local school database, or separately maintained frontend.

From the repository root:

```sh
# Install workspace dependencies first (as described in the root README).
npm run build --workspace @edupilot/web
node clients/desktop/scripts/build.mjs
EDUPILOT_API_ORIGIN=https://school.example.org clients/desktop/bin/edupilot
```

The script copies `clients/web/dist/` byte-for-byte into `clients/desktop/frontend/dist/` before compiling. Keep the web API base at its default relative `/api/v1` for this build. Commit source, not the copied dist or native binary. If no origin is set, the app connects to a local development server at `http://127.0.0.1:3000`. Use HTTPS for deployed schools. The setting is an origin (scheme + host + optional port); paths, embedded credentials, and non-loopback HTTP are rejected. Windows writes `clients/desktop/bin/edupilot.exe`; set the environment variable before starting it.

A bounded, same-origin asset handler forwards only `/api/v1/` to that single origin. It passes bearer credentials and content negotiation headers, discards cookies and school overrides, and rejects redirects. The backend still resolves the school, checks grants and persists data. On a connection error after a write, refresh the register before resubmitting.

**Documents.** The desktop app forwards raw uploads to `POST /api/v1/documents/files` through 8 MiB (8,388,608 bytes), the same maximum as the API. The 201 response carries the file ID and can include a validated local download path. Downloading `GET /api/v1/documents/files/{id}` returns the original bytes with the API’s attachment filename and allowlisted media type (PDF, PNG, JPEG or plain text). The gateway requires the filename to match the requested UUID and file type. Error responses remain JSON. Other API requests retain their 1 MiB limit.

**Native prerequisites:** Go 1.26+, a native C toolchain and the dependencies described in [Wails installation](https://wails.io/quick-start/installation/). On Linux install GTK 4 and WebKitGTK 6 development packages and run in a graphical session; macOS needs Xcode command-line tools, Windows needs WebView2. This script builds a local executable for the host OS. Distribution installers, platform signing and notarization require your release identities.

The native Go module is deliberately separate from the Goravel server. For an assets-only preparation use `node clients/desktop/scripts/build.mjs --assets-only` after the web build. Run gateway tests in this directory with `go test ./internal/gateway`. Native CI also builds the Linux desktop with its web assets, checks the branded window on a disposable X11 display, and cross-compiles the Windows executable. Windows interaction, macOS builds, installers and signing have separate release checks.

Sources: [Wails asset routing](https://v3.wails.io/contributing/asset-server/) · [versioned Go API](https://pkg.go.dev/github.com/wailsapp/wails/v3@v3.1.0/pkg/application).
