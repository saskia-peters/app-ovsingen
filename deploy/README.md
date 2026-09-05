# deploy

Deployment definitions + backup configuration. The local development stack
is `compose.yaml` at the repo root (driven by `just db-up`/`just dev`); this
directory is reserved for later stories/deploy epic:

- production compose variants for the self-hosted single host,
- edge / reverse proxy and CDN integration (Cloudflare or bunny.net for TLS termination, caching, and DDoS mitigation),
- backup job definitions and restore procedure (NFR-R3),
- environment-specific compose overrides.

**No docker-specific config; the Compose source is driven via podman-compose.**
