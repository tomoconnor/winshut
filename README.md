# WinShut

Remote Windows power management over HTTPS. Shutdown, hibernate, or sleep a Windows machine via a simple API, secured with mTLS and/or bearer token auth.

Built because OpenSSH on Windows is unreliable for remote power management.

## Build

**macOS cross-compile (produces Windows binary):**

```bash
make build-windows
```

**Native build (stub power commands on non-Windows):**

```bash
make build
```

## Certificate Generation

WinShut requires TLS. For production use, generate a CA, server cert, and optionally client certs for mTLS.

**Dev certs (CA + server + client):**

```bash
make dev-certs
```

You'll be prompted for SANs (comma-separated hostnames and IPs for the server cert):

```
Enter SANs (comma-separated hostnames/IPs, e.g. mypc.local,192.168.1.100): mypc.local,192.168.1.100
```

This generates all files in `certs/`:
- `ca.crt` / `ca.key` - CA certificate and key
- `server.crt` / `server.key` - server TLS cert with your SANs
- `client.crt` / `client.key` - client cert for mTLS

**Manual production certs:**

```bash
# CA
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
  -keyout ca.key -out ca.crt -days 3650 -nodes \
  -subj "/CN=WinShut CA"

# Server cert (set SANs to your machine's hostname/IP)
openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
  -keyout server.key -out server.csr -nodes \
  -subj "/CN=mypc.local"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 365 \
  -extfile <(printf "subjectAltName=DNS:mypc.local,IP:192.168.1.100")

# Client cert (for mTLS)
openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
  -keyout client.key -out client.csr -nodes \
  -subj "/CN=winshut-client"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client.crt -days 365
```

## Usage

```
winshut --cert server.crt --key server.key [options]

Options:
  --addr     Listen address (default: :9090)
  --cert     TLS certificate file (required)
  --key      TLS private key file (required)
  --ca       CA cert for mTLS client verification
  --token    Bearer token for Authorization header auth
  --dry-run  Log commands without executing
```

At least one of `--ca` or `--token` must be provided (fail-closed).

**Run locally for development:**

```bash
make dev
```

## API

| Method | Path         | Auth | Description                |
|--------|--------------|------|----------------------------|
| POST   | `/shutdown`  | Yes  | Immediate shutdown         |
| POST   | `/hibernate` | Yes  | Hibernate                  |
| POST   | `/sleep`     | Yes  | Sleep (suspend to RAM)     |
| GET    | `/health`    | No   | Liveness check             |
| GET    | `/stats`     | No   | CPU and memory statistics   |

All power endpoints return a JSON response before executing the command (500ms delay).

## CLI Client

A cross-platform CLI client for interacting with the winshut server from Linux/macOS.

**Build:**

```bash
make build-client
```

**Config file** (`winshut-client.yml` in current directory by default):

```yaml
server: https://mypc.local:9090
ca: certs/ca.crt
cert: certs/client.crt
key: certs/client.key
token: SECRET
```

- `server` — required, base URL of the winshut server
- `ca` — optional, CA cert for server verification (uses system roots if omitted)
- `cert` / `key` — optional, client cert pair for mTLS
- `token` — optional, bearer token for auth on power commands

**Usage:**

```bash
# Health check
./winshut-client health

# System stats
./winshut-client stats

# Power commands (require auth via mTLS and/or token)
./winshut-client shutdown
./winshut-client hibernate
./winshut-client sleep

# Custom config path
./winshut-client --config /path/to/config.yml health
```

## curl Examples

All examples use `--cacert` to verify the server's TLS certificate against your CA. Without it, you'd need `-k` to skip verification.

**Bearer token auth:**

```bash
curl --cacert certs/ca.crt -X POST \
  -H "Authorization: Bearer SECRET" \
  https://localhost:9090/shutdown
```

**mTLS auth (no token needed):**

```bash
curl --cacert certs/ca.crt \
  --cert certs/client.crt --key certs/client.key \
  -X POST https://localhost:9090/shutdown
```

**Health check (no auth required):**

```bash
curl --cacert certs/ca.crt https://localhost:9090/health
```

**System stats (no auth required):**

```bash
curl --cacert certs/ca.crt https://localhost:9090/stats
```

Returns:

```json
{"cpu_usage_percent":12,"memory_total_bytes":17179869184,"memory_free_bytes":8589934592,"memory_used_bytes":8589934592}
```

**Response format (power endpoints):**

```json
{"status":"ok","action":"shutdown","message":"executing"}
```

## Windows Setup

### Firewall Rule

```powershell
New-NetFirewallRule -DisplayName "WinShut" -Direction Inbound -LocalPort 9090 -Protocol TCP -Action Allow
```

### Auto-Start with Task Scheduler

```
schtasks /create /sc onstart /tn "WinShut" /tr "C:\winshut\winshut.exe --cert C:\winshut\server.crt --key C:\winshut\server.key --ca C:\winshut\ca.crt --token SECRET" /ru SYSTEM /rl HIGHEST
```

Place `winshut.exe` and cert files in `C:\winshut\` (or adjust paths).
