# acme-challenge-outpost

Small ACME challenge outpost for services not exposed to the internet.

- Public server: answers only `/.well-known/acme-challenge/{token}` from memory.
- Admin server (IP allow-listed, path ignored; `/metrics` serves Prometheus metrics):
  - `?mode=create&token={token}&keyauthorization={keyauth}`
  - `?mode=clean&token={token}`

Both servers honor `X-Forwarded-*` from `trustedProxyIPs` and write zerolog access logs.

Run: `go run . -config config.yaml` (see `config.example.yaml`).
