# 0005 — WEBUI presence, relay and setup page

## Status

Accepted

## Date

2026-09-01 (presence + relay, commit `cbcdecaf`; CSL relay in `1755ae81`;
password-reset helper flow added 2026-10-04 in `4ae8a10`/`b4a041ec`)

## Context

The HRPAuth WEBUI dashboard aggregates microservice pages through a
"presence" registry: a service announces itself, declares which frontend
areas it covers, and registers relay rules so browsers can reach it through
the main service origin. HASkinProxy contributes a CustomSkinLoader setup
page to the dashboard.

## Decision (implemented state on `main`)

1. **Presence handshake** (non-blocking goroutine at startup):
   `POST {base}/services/presence` with name (default `HASkinProxy`),
   optional TTL, scope `{name: haskinproxy, frontend_areas: [webui-dash]}`,
   and `sdk_url`. The `sdk_url` is built on the **main service callback
   origin** (resolved via `GET {base}/status` → `backend.url`), never on
   the proxy's internal address; fallback is `server.public_url`.
2. **Relay rules**: `POST {base}/services/relay` registers
   - `/sdk/haskinproxy.js` → proxy's SDK JS (HRPAuth fetches it via relay),
   - `/customskinloader` → the setup page (iframe in the dashboard),
   - `/auth` → password-reset helper endpoints,
   - `/csl` → the proxy root (CSL API for Minecraft clients; prefix
     stripped by the relay).
3. **Runtime SDK mechanism**: `GET /sdk/haskinproxy.js` returns a small JS
   object (`window['HASkinProxy-sdk']`) declaring the dashboard menu item
   whose URL is `{__BACKEND_URL__}/customskinloader`.
4. **Setup page**: `GET /customskinloader` renders a self-contained HTML
   page with a CSL `config.json` generator (API root derived from the main
   service callback URL + `/csl/`) and usage instructions.
5. **Helper auth flow** for the page: `/auth/captcha`,
   `/auth/send-reset-code`, `/auth/verify-reset-code` — graphic captcha +
   4-digit email code (TTL `auth.code_ttl`, default 600 s), sent via the
   configured SMTP. This is account recovery for the *setup page*, not
   authentication of the CSL surface.
6. Presence/relay failures only log warnings; the proxy keeps serving CSL.

## Consequences

- Benefit: zero manual WEBUI configuration; the dashboard entry appears
  when the handshake succeeds.
- Cost: the handshake depends on the management surface of the same
  `base_url` that [0004](./0004-upstream-drift-and-dead-calls.md) shows to
  be in conflict with the Yggdrasil surface — on `main`, getting the CSL
  flow to work typically breaks the handshake (and vice versa).
- Cost: SMTP and captcha machinery live in this proxy (duplicated concern)
  solely for the setup page's reset flow.
- Branch `origin/10426` replaces items 3-4 with a compile-time SDK package
  (`sdk/` embedded, uploaded via `POST /services/sdk-packages`) and drops
  the runtime WEBUI/auth routes; HA-Contract's dev doc already describes
  that design. Until merged, `main` behavior is what operators get.
  **Update 2026-11-05**: the mainline decision (entry
  [0007](./0007-mainline-decision.md)) made `10426` the main branch and
  retired `main` (archived as tag `archive/main-2026-11-04`). The
  compile-time SDK design is now what operators get; the runtime WEBUI
  behavior described in this entry (items 3-5, including the password-reset
  helper flow) exists only in the archived `main` tree.

## Evidence

- Presence + relay registration: `main.go:L65-L127 @ b4a041ec` (sdk_url
  from callback at `main.go:L78-L105 @ b4a041ec`; relay rules at
  `main.go:L110-L121 @ b4a041ec`), HEAD of `main` `b4a041ec`.
- Callback URL resolution: `internal/hrpauth/client.go:L229-L276 @ b4a041ec`
  (`GET /status` → `backend.url`).
- Presence/relay client calls: `internal/hrpauth/client.go:L311-L334 @ b4a041ec`
  (`/services/presence`), `internal/hrpauth/client.go:L352-L378 @ b4a041ec`
  (`/services/relay`).
- SDK JS content and content type: `internal/handler/webui_handler.go:L9-L46 @ b4a041ec`.
- Setup page handler: `internal/handler/webui_handler.go:L48-L55 @ b4a041ec`
  (API root derivation described at
  `internal/handler/webui_handler.go:L49-L51 @ b4a041ec`).
- Auth helper routes: `internal/router/router.go:L24-L29 @ b4a041ec`;
  handlers `internal/handler/auth_handler.go:L11-L103 @ b4a041ec`; service
  `internal/service/auth_service.go:L24-L135 @ b4a041ec` (captcha
  `internal/service/auth_service.go:L32-L43 @ b4a041ec`, reset code
  `internal/service/auth_service.go:L45-L130 @ b4a041ec`).
- SMTP / code TTL config: `config/config.go:L38-L49 @ b4a041ec`,
  `config/config.go:L112-L116 @ b4a041ec`.
- Commits: presence `cbcdecaf69c91ed227f17822278fdcd084bb2f57` (2026-09-01),
  CSL relay `1755ae81c39a9c151dc53c770dc2beea72f887cf` (2026-09-01),
  reset flow `4ae8a10cba9db050ac615513a933c6edea014de5` + `b4a041ec` (HEAD
  of `main`).
- Contract-side description of the newer SDK mechanism: repo HA-Contract
  (@ `21c7147d`),
  `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HASkinProxy/README.md`
  ("shipped as a compile-time SDK package").
- Contract auth model for these management calls (repo HA-Contract
  @ `21c7147d`):
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`
  (microservice→core calls must use a Bearer token from the
  `client_credentials` grant),
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L97-L118 @ 21c7147d`
  (temporary internal key via `POST /services/auth-key`,
  authenticated with client-credentials Bearer). The `main`-branch static
  bearer therefore also falls short of the contract here — see entry
  [0004](./0004-upstream-drift-and-dead-calls.md) item 4.
