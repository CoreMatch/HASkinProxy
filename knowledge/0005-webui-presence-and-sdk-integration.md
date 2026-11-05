# 0005 — WEBUI presence, relay and SDK package

## Status

Accepted

## Date

2026-09-01 (presence + relay, commit `cbcdecaf`; CSL relay in `1755ae81`;
compile-time SDK package 2026-10-09 in `6830c9c`)

## Context

The HRPAuth WEBUI dashboard aggregates microservice pages through a
"presence" registry: a service announces itself, declares which frontend
areas it covers, and registers relay rules so browsers can reach it through
the main service origin. HASkinProxy contributes a CustomSkinLoader setup
page to the dashboard, delivered as a compile-time SDK package.

## Decision

1. **Presence handshake** (non-blocking goroutine at startup):
   `POST {base}/services/presence` with name (default `HASkinProxy`),
   optional TTL (`ttl_seconds`), and scope
   `{name: haskinproxy, frontend_areas: [webui-dash]}`.
2. **Relay rule**: `POST {base}/services/relay` registers `/csl` → the
   proxy's `server.public_url` (CSL API for Minecraft clients; prefix
   stripped by the relay).
3. **Compile-time SDK package**: the setup page is the embedded `sdk/`
   source tree (manifest.json + React page). At startup the proxy packs it
   into a tar.gz archive and uploads it via `POST /services/sdk-packages`;
   on a 409 conflict it deletes the stale package and retries once, so
   every restart converges the remote package to the embedded sources. The
   WEBUI SDK handler aggregates the package into the frontend build; no
   runtime page route is served.
4. Presence/relay/SDK-upload failures only log warnings; the proxy keeps
   serving CSL.

## Consequences

- Benefit: zero manual WEBUI configuration; the dashboard entry appears
  when the handshake succeeds.
- The relay rule requires `server.public_url` to be reachable from
  HRPAuth; with an empty `public_url` only the presence handshake runs.
- The dashboard page always matches the running binary, since the package
  is re-converged on every restart.
- The management calls in this entry authenticate through the OAuth2
  client-credentials token — see entry
  [0004](./0004-upstream-drift-and-dead-calls.md) item 3.

## Evidence

- Presence + relay registration: `main.go::announcePresence` (scope and
  the `/csl` relay rule).
- Presence request shape: `internal/hrpauth/client.go::PresenceRequest`.
- Relay client call: `internal/hrpauth/client.go::RegisterRelay`.
- SDK embed: `sdkdata.go::sdkSource`.
- SDK archive + upload flow: `sdk_upload.go::uploadSDKPackage`
  (delete + retry on conflict), `sdk_upload.go::buildSDKArchive`.
- SDK package client calls: `internal/hrpauth/client.go::UploadSDKPackage`,
  `internal/hrpauth/client.go::DeleteSDKPackage`.
- Presence/SDK config: `config/config.go::PresenceConfig`,
  `config/config.go::SDKConfig`, `config/config.go (field: public_url)`.
- Commits: presence `cbcdecaf69c91ed227f17822278fdcd084bb2f57`
  (2026-09-01), CSL relay
  `1755ae81c39a9c151dc53c770dc2beea72f887cf` (2026-09-01), compile-time
  SDK package `6830c9c41c20928f41631c6d6d78abac5cdb53d0` (2026-10-09).
- Contract-side description of the SDK mechanism: repo HA-Contract
  (@ `21c7147d`),
  `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HASkinProxy/README.md`
  ("shipped as a compile-time SDK package").
