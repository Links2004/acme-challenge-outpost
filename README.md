# acme-challenge-outpost

Small ACME challenge outpost for services not exposed to the internet.

- Public server: answers only `/.well-known/acme-challenge/{token}` from memory.
- Admin server (IP allow-listed, path ignored; `/metrics` serves Prometheus metrics):
  - `?mode=create&token={token}&keyauthorization={keyauth}`
  - `?mode=clean&token={token}`

Both servers honor `X-Forwarded-*` from `trustedProxyIPs` and write zerolog access logs.

Run: `go run . -config config.yaml` (see `config.example.yaml`).

## Docker / Kubernetes

CI publishes multi-arch (linux/amd64, linux/arm64) distroless images to
`ghcr.io/links2004/acme-challenge-outpost` and attaches linux binaries to
tagged releases. The image runs as non-root and reads
`/etc/acme-challenge-outpost/config.yaml` (mount it from a ConfigMap).

## Graceful shutdown

On `SIGTERM` (Kubernetes pod termination, `systemctl stop`) or `SIGINT` both
servers stop accepting connections and wait up to 20s for in-flight requests.
Challenges are held in memory only, so they are lost on restart.

Example systemd unit:

```ini
[Service]
ExecStart=/usr/local/bin/acme-challenge-outpost -config /etc/acme-challenge-outpost/config.yaml
KillSignal=SIGTERM
TimeoutStopSec=30
Restart=on-failure
```
