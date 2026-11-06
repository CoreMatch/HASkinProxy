# 0005 — WEBUI presence, relay and SDK package

## Status

Accepted

## Context

The HRPAuth WEBUI dashboard aggregates microservice pages through a
"presence" registry: a service announces itself, declares which frontend
areas it covers, and registers relay rules so browsers can reach it through
the main service origin. HASkinProxy contributes a CustomSkinLoader setup
page to the dashboard, delivered as a compile-time SDK package.

## Decision

1. **Presence handshake**: a synchronous token-less bootstrap call
   (`internal/hrpauth/selfregister.go::RegisterPresencePlain`, required
   before credential resolution, item 5) followed by the asynchronous
   goroutine at startup: `POST {base}/services/presence` with name
   (default `HASkinProxy`), optional TTL (`ttl_seconds`), and scope
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
5. **Credential resolution** (`main.go::resolveUpstreamCredentials`): the
   configured `upstream` credentials are classified at startup via
   HRPAuth's RFC 7591 protected registration endpoint
   `POST /oauth/register`
   (`internal/hrpauth/selfregister.go::ResolveServiceCredentials`), which
   only authenticates the built-in super client —
   - `201`: super credentials; exchanged for this service's own
     low-privilege client (`client_id` = presence name, stable secret via
     HRPAuth get-or-create). The super secret is discarded from memory
     and the config file rewritten
     (`config/configwrite.go::RewriteClientCredentials`: atomic temp-file
     + rename, `.bak`, 0600, re-parse validation, comment-preserving
     yaml.Node surgery).
   - `400 invalid_client_metadata`: super credentials but registration
     rejected (presence missing, scope not allowed) — startup aborts.
   - `401`: not super; a plain `client_credentials` token request
     verifies the pair is an ordinary service client, reused as-is
     (idempotent path — no exchange, no write-back).
   Any resolution failure aborts startup (`log.Fatalf`): super
   credentials must never reach business requests. Entrypoint-rendered
   deployments (`config/configwrite.go::IsRenderedDeployment`:
   `CONFIG_TEMPLATE` + `RENDERED_CONFIG`, the HA container convention)
   are only warned about — their config is re-rendered every start, so
   the template/environment side must carry the credentials; a failed
   write-back is likewise non-fatal (ERROR log, next startup exchanges
   again).

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
- Credential resolution makes HRPAuth reachability a startup requirement
  whenever credentials are configured; the bootstrap presence and the
  register call themselves carry no Bearer token (the super secret, when
  present, is used for the exchange call only).
- HASkinProxy needs no specific OAuth scopes: every endpoint it calls is
  either unauthenticated, ops-level (service-token subject type, scope
  list not checked) or Yggdrasil-plane (OAuth tokens ignored), so its
  HRPAuth-side scope allowlist entry is correctly empty.

## Evidence

- Presence + relay registration: `main.go::announcePresence` (scope and
  the `/csl` relay rule).
- Credential resolution / super exchange:
  `internal/hrpauth/selfregister.go::ResolveServiceCredentials`,
  `internal/hrpauth/selfregister.go::RegisterPresencePlain`; startup
  wiring `main.go::resolveUpstreamCredentials`; write-back
  `config/configwrite.go::RewriteClientCredentials`, rendered-deployment
  detection `config/configwrite.go::IsRenderedDeployment` (env
  convention set by `docker-entrypoint.sh`).
- Presence request shape: `internal/hrpauth/client.go::PresenceRequest`.
- Relay client call: `internal/hrpauth/client.go::RegisterRelay`.
- SDK embed: `sdkdata.go::sdkSource`.
- SDK archive + upload flow: `sdk_upload.go::uploadSDKPackage`
  (delete + retry on conflict), `sdk_upload.go::buildSDKArchive`.
- SDK package client calls: `internal/hrpauth/client.go::UploadSDKPackage`,
  `internal/hrpauth/client.go::DeleteSDKPackage`.
- Presence/SDK config: `config/config.go::PresenceConfig`,
  `config/config.go::SDKConfig`, `config/config.go (field: public_url)`.
- Commits: presence `cbcdecaf69c91ed227f17822278fdcd084bb2f57`,
  CSL relay `1755ae81c39a9c151dc53c770dc2beea72f887cf`, compile-time
  SDK package `6830c9c41c20928f41631c6d6d78abac5cdb53d0`.
- Contract-side description of the SDK mechanism: repo HA-Contract
  (@ `21c7147d`),
  `/home/lnb/Desktop/HA/HASkinProxy/knowledge/0008-service-development-guide.md`
  ("shipped as a compile-time SDK package").
