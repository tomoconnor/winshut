# WinShut AI coding instructions

## Big picture
- Single Go service in main package; server setup in [main.go](main.go) with `http.ServeMux` and TLS/mTLS config.
- HTTP handlers live in [handlers.go](handlers.go); auth logic in [auth.go](auth.go).
- Power and stats are OS-specific via build tags: [power_windows.go](power_windows.go), [stats_windows.go](stats_windows.go) for Windows, with non-Windows stubs in [power_stub.go](power_stub.go) and [stats_stub.go](stats_stub.go).
- CLI client is a separate command in [cmd/winshut-client/main.go](cmd/winshut-client/main.go), configured by YAML.

## Runtime behavior and API
- Server requires TLS; `--cert` and `--key` are mandatory. It fails closed unless at least one of `--ca` (mTLS) or `--token` is set (see [main.go](main.go)).
- Auth accepts either a verified client cert (`r.TLS.VerifiedChains`) OR a bearer token (`Authorization: Bearer ...`) via [auth.go](auth.go).
- Power endpoints (`/shutdown`, `/hibernate`, `/sleep`) are POST only and return JSON before executing the command; actual execution is delayed 500ms in a goroutine (see [handlers.go](handlers.go)).
- `/health` and `/stats` are GET-only and unauthenticated.
- Windows stats use `wmic` commands (CPU/memory/uptime) in [stats_windows.go](stats_windows.go); non-Windows uses runtime memory + process uptime in [stats_stub.go](stats_stub.go).

## Cross-component conventions
- Client commands map to HTTP paths in [cmd/winshut-client/main.go](cmd/winshut-client/main.go). If you add/rename a server route, update the client `commands` map and README examples together.
- CLI config is YAML (`winshut-client.yml` by default) with keys `server`, `ca`, `cert`, `key`, `token`.

## Developer workflows
- Build server: `make build` (native), `make build-windows` (cross-compile).
- Build client: `make build-client`.
- Dev TLS certs: `make dev-certs` (writes certs/ca.crt, server.crt, client.crt, etc.).
- Run locally in dry-run mode: `make dev`.
- Bundle release artifacts: `make package` (zip with binaries + certs).

## Integration points and dependencies
- External commands on Windows: `shutdown`, `rundll32`, `powershell`, `wmic` (see [power_windows.go](power_windows.go), [stats_windows.go](stats_windows.go)).
- YAML parsing uses `gopkg.in/yaml.v3` in the client.
