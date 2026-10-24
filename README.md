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
