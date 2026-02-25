# WinShut

Remote Windows power management over HTTPS. Shutdown, restart, hibernate, sleep, lock, logoff, or turn off the screen on a Windows machine via a simple API, secured with mTLS.

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

**CLI client (Linux/macOS):**

```bash
make build-client
```

## Certificate Generation

WinShut requires mTLS. You need a CA, server cert, and client cert.

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

## Server Usage

```
winshut --cert server.crt --key server.key --ca ca.crt [options]

Options:
  --addr     Listen address (default: :9090)
  --cert     TLS certificate file (required)
  --key      TLS private key file (required)
  --ca       CA cert for mTLS client verification (required)
  --dry-run  Log commands without executing
```

**Run locally for development:**

```bash
make dev
```

## API

| Method | Path          | Auth | Description                |
|--------|---------------|------|----------------------------|
| GET    | `/health`     | No   | Liveness check             |
| GET    | `/stats`      | No   | CPU, memory, and uptime    |
| POST   | `/shutdown`   | mTLS | Immediate shutdown         |
| POST   | `/restart`    | mTLS | Immediate restart          |
| POST   | `/hibernate`  | mTLS | Hibernate                  |
| POST   | `/sleep`      | mTLS | Sleep (suspend to RAM)     |
| POST   | `/lock`       | mTLS | Lock workstation           |
| POST   | `/logoff`     | mTLS | Log off current user       |
| POST   | `/screen-off` | mTLS | Turn off monitor(s)        |

All power endpoints return a JSON response before executing the command (500ms delay).

## CLI Client

A cross-platform CLI client for interacting with the winshut server from Linux/macOS.

**Config file** (`winshut-client.yml` in current directory by default):

```yaml
server: https://mypc.local:9090
ca: certs/ca.crt
cert: certs/client.crt
key: certs/client.key
```

- `server` — required, base URL of the winshut server
- `ca` — optional, CA cert for server verification (uses system roots if omitted)
- `cert` / `key` — required, client cert pair for mTLS

The client warns if the config file has loose permissions (should be `chmod 600`).

**Usage:**

```bash
./winshut-client health
./winshut-client stats
./winshut-client shutdown
./winshut-client restart
./winshut-client hibernate
./winshut-client sleep
./winshut-client lock
./winshut-client logoff
./winshut-client screen-off

# Custom config path
./winshut-client --config /path/to/config.yml health
```

## curl Examples

All examples use `--cacert` to verify the server's TLS certificate and `--cert`/`--key` for mTLS client authentication.

**Shutdown:**

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
{"cpu_usage_percent":12,"memory_total_bytes":17179869184,"memory_free_bytes":8589934592,"memory_used_bytes":8589934592,"uptime_seconds":86400}
```

**Response format (power endpoints):**

```json
{"status":"ok","action":"shutdown","message":"executing"}
```

## Packaging

**Safe package (public certs only):**

```bash
make package
```

**Full package with private keys (for deployment to target machine):**

```bash
make package-insecure
```

## Windows Setup

### Firewall Rule

```powershell
New-NetFirewallRule -DisplayName "WinShut" -Direction Inbound -LocalPort 9090 -Protocol TCP -Action Allow
```

### Auto-Start with Task Scheduler

```
schtasks /create /sc onstart /tn "WinShut" /tr "C:\winshut\winshut.exe --cert C:\winshut\server.crt --key C:\winshut\server.key --ca C:\winshut\ca.crt" /ru SYSTEM /rl HIGHEST
```

Place `winshut.exe` and cert files in `C:\winshut\` (or adjust paths).

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
