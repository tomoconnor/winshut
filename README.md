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

**All client platforms (linux/darwin/windows, amd64/arm64):**

```bash
make build-client-all
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
- `server.crt` / `server.key` - server TLS cert with your SANs (EKU: serverAuth)
- `client.crt` / `client.key` - client cert for mTLS (EKU: clientAuth)

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

cat > server.cnf <<EOF
[server_ext]
keyUsage = digitalSignature
extendedKeyUsage = serverAuth
subjectAltName = DNS:mypc.local,IP:192.168.1.100
EOF

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out server.crt -days 365 \
  -extensions server_ext -extfile server.cnf

# Client cert (for mTLS)
openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
  -keyout client.key -out client.csr -nodes \
  -subj "/CN=winshut-client"

cat > client.cnf <<EOF
[client_ext]
keyUsage = digitalSignature
extendedKeyUsage = clientAuth
EOF

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client.crt -days 365 \
  -extensions client_ext -extfile client.cnf

# Clean up
rm -f server.csr client.csr server.cnf client.cnf ca.srl
```

## Server Usage

```
winshut [command] [options]

Commands:
  install    Install as a Windows service (Windows only)
  remove     Remove the Windows service (Windows only)

Options:
  --addr     Listen address (default: :9090)
  --cert     TLS certificate file (required)
  --key      TLS private key file (required)
  --ca       CA cert for mTLS client verification (required)
  --dry-run  Log commands without executing
```

Use `--addr` to bind to a specific interface, e.g. `--addr 192.168.1.100:9090`.

**Run locally for development:**

```bash
make dev
```

## API

All endpoints require a valid mTLS client certificate.

| Method | Path          | Description                |
|--------|---------------|----------------------------|
| GET    | `/health`     | Liveness check             |
| GET    | `/stats`      | CPU, memory, and uptime    |
| POST   | `/shutdown`   | Immediate shutdown         |
| POST   | `/restart`    | Immediate restart          |
| POST   | `/hibernate`  | Hibernate                  |
| POST   | `/sleep`      | Sleep (suspend to RAM)     |
| POST   | `/lock`       | Lock workstation           |
| POST   | `/logoff`     | Log off current user       |
| POST   | `/screen-off` | Turn off monitor(s)        |

All power endpoints return a JSON response before executing the command (500ms delay).

## CLI Client

A cross-platform CLI client for interacting with the winshut server.

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

All examples require `--cacert` for server verification and `--cert`/`--key` for mTLS client authentication.

**Shutdown:**

```bash
curl --cacert certs/ca.crt \
  --cert certs/client.crt --key certs/client.key \
  -X POST https://mypc.local:9090/shutdown
```

**Health check:**

```bash
curl --cacert certs/ca.crt \
  --cert certs/client.crt --key certs/client.key \
  https://mypc.local:9090/health
```

**System stats:**

```bash
curl --cacert certs/ca.crt \
  --cert certs/client.crt --key certs/client.key \
  https://mypc.local:9090/stats
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

### Windows Defender Exclusion

Unsigned Go binaries may trigger a false positive in Windows Defender. Exclude the install directory:

```powershell
Add-MpPreference -ExclusionPath "C:\winshut"
```

### Firewall Rule

```powershell
New-NetFirewallRule -DisplayName "WinShut" -Direction Inbound -LocalPort 9090 -Protocol TCP -Action Allow
```

### Windows Service

WinShut can run as a native Windows service with Event Viewer logging.

**Install the service:**

```powershell
winshut.exe install --cert C:\winshut\server.crt --key C:\winshut\server.key --ca C:\winshut\ca.crt
```

Flags passed after `install` are stored as the service's startup arguments. The service is configured to start automatically on boot.

**Start / stop:**

```powershell
sc start WinShut
sc stop WinShut
```

**Remove the service:**

```powershell
winshut.exe remove
```

**Event Viewer:** Logs appear in the Application log under source "WinShut".

Place `winshut.exe` and cert files in `C:\winshut\` (or adjust paths).

**Note:** Running as SYSTEM works for shutdown, restart, hibernate, sleep, and screen-off. However, `/lock` and `/logoff` affect the interactive console session and may not work correctly from a SYSTEM service. If you need those commands, run winshut under the interactive user account instead of SYSTEM.

## Certificate Rotation

Dev certs generated by `make dev-certs` expire after 365 days (CA after 10 years). To rotate:

1. Re-run `make dev-certs` to generate a fresh set
2. Copy the new `server.crt` and `server.key` to the Windows machine and restart winshut
3. Copy the new `client.crt`, `client.key`, and `ca.crt` to your client machine

The server reads certs at startup, so it must be restarted to pick up new certs. The CA key is only needed for signing — it doesn't need to be on the server or client machines in production.

## License

This project is licensed under the [Mozilla Public License 2.0](LICENSE).
