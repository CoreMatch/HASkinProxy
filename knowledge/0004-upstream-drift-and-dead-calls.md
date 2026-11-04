# 0004 — Upstream drift and dead calls

## Status

Accepted (documents the implemented state on `main`; the drift findings are
verified against the upstream repositories as of 2026-11-04)

## Date

2026-11-04

## Context

HASkinProxy was written against an upstream layout where one base URL
covered both the Yggdrasil endpoints and the HRPAuth management endpoints.
The HA stack since split the Yggdrasil surface into the
HRPAuth-Yggdrasil-API microservice, reachable through HRPAuth's relay under
the `/yggdrasil-api` prefix (prefix stripped when forwarding). Several
outbound calls on `main` no longer succeed against the current upstream.

## Decision (implemented state)

Recorded drift items, each verified:

1. **Delete passthrough is dead (404).**
   `POST /{username}/texture/delete` forwards the body unchanged to
   `POST {upstream.base_url}/texture/delete`. The current
   HRPAuth-Yggdrasil-API registers no such route; it exposes
   `DELETE /texture/delete/{uuid}/{type}` instead. The forwarded call
   therefore receives a 404 (or hits an unrelated route depending on the
   configured base URL). The handler comment still lists legacy body fields
   (`remember_token, profile_id, texture_type, auth_type, uid, email`) that
   match neither the current Yggdrasil-API nor the current HASkinLib delete
   API (multipart `type`+`hash` with OAuth2 Bearer). HA-Contract documents
   this endpoint as legacy-only and states it 404s against the current
   upstream.
2. **Single `base_url` cannot satisfy both surfaces.**
   All outbound URLs are built as `{upstream.base_url} + path`:
   - Yggdrasil calls (`/api/profiles/minecraft`,
     `/sessionserver/session/minecraft/profile/{uuid}`, `/textures/{hash}`,
     `/texture/delete`) only resolve when the base URL includes the
     `/yggdrasil-api` relay prefix (HRPAuth root has none of these routes).
   - Management calls (`/services/presence`, `/services/relay`, `/status`,
     `/api/user?email=`) only resolve against the HRPAuth **root** (the
     relay serves `/yggdrasil-api/*` only and skips `/services/*`).
   With the default base URL (`http://localhost`, port `2778`, the root),
   the CSL core flow 404s; with a `/yggdrasil-api` base URL, presence/relay
   registration and the email check 404 instead. There is no configuration
   on `main` that makes everything work.
3. **Email existence check uses a non-existent endpoint.**
   `GET {base}/api/user?email={email}` — HRPAuth root registers
   `POST /user` (no GET variant, no email query form, and no `/api/user`
   route at all). The reset-code flow treats any non-200 as "user not
   found", so the check degrades to always-false rather than erroring.
4. **Auth style is a static bearer — the contract does not allow it.**
   Service-side requests send `Authorization: Bearer {client_secret}` plus
   `X-Client-ID` instead of an OAuth2 client-credentials token exchange.
   Checked against the contract (2026-11-04):
   - [xref] HA-Contract `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L30 @ 21c7147d`
     — the "Built-in Super
     Client Secret" (`client_secret`) is a credential **for
     `client_credentials`**, and its listed usage endpoint is only
     `POST /oauth/token (grant_type=client_credentials)`.
   - [xref] HA-Contract `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`
     — "Used for all
     microservice -> HRPAuth Core API calls. Microservices must provide a
     Bearer token obtained via the `client_credentials` grant."
   - [xref] HA-Contract `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L382 @ 21c7147d`
     — auth-level
     resolution: "no `Authorization: Bearer` → `0`; token equals M-T →
     `2`; resolvable OAuth2 token → `2` if service token, else `1`;
     unresolvable → `0`." A raw `client_secret` is not an issued token and
     is not the M-T, so it resolves to level 0.
   - The only static bearer the contract accepts is the legacy Maintenance
     Token ([xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L38 @ 21c7147d`,
     [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L380 @ 21c7147d`),
     explicitly marked legacy
     ([xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L42 @ 21c7147d`).
   Conclusion: presenting `client_secret` directly as a Bearer is **not
   sanctioned by the contract**; management calls that need auth level ≥ 1
   will fail or be treated as anonymous. (Whether the running HRPAuth
   build additionally tolerates it is an implementation question the
   contract does not cover — not checked here.)

## Consequences

- Against the current HA deployment topology, `main` provides a working
  setup only if operators accept partial breakage (typically: CSL flow
  works with a `/yggdrasil-api` base URL; presence/relay/WEBUI integration
  and delete do not).
- Branch `origin/10426` (`a83de4d2`) partially fixes items 1-2 and 4:
  separate Yggdrasil base (`base_url + /yggdrasil-api`), OAuth2
  client-credentials token exchange, compile-time SDK package replacing the
  runtime WEBUI surface. It is pushed but **not merged** into `main` at
  writing time; HA-Contract's dev doc already describes that newer design.
  Merging it is the intended path; until then this entry is the operator
  reference.
  **Update 2026-11-05**: the mainline decision (entry
  [0007](./0007-mainline-decision.md)) made `10426` the main branch;
  `main` was retired and archived as tag `archive/main-2026-11-04`. The
  fixes described above are now what operators get from the mainline, and
  the "pushed but not merged" state no longer exists. Item 3 is moot on
  `10426`: the setup-page auth flow that performed the email check was
  removed entirely (compile-time SDK package, commit `6830c9c`).
- The delete passthrough should be treated as compatibility-only for
  legacy upstreams (per HA-Contract); do not build user-facing flows on it.

## Evidence

- Delete passthrough target and verbatim forwarding:
  `internal/hrpauth/client.go:L134-L170 @ b4a041ec` (URL at
  `internal/hrpauth/client.go:L147 @ b4a041ec`, JSON content type at
  `internal/hrpauth/client.go:L153 @ b4a041ec`),
  `internal/handler/csl_handler.go:L138-L194 @ b4a041ec` (legacy field
  list at `internal/handler/csl_handler.go:L144-L146 @ b4a041ec`),
  HEAD of `main` `b4a041ec`.
- Upstream route reality (repo HRPAuth-Yggdrasil-API @ `124b2fa1`):
  [xref] `/home/lnb/Desktop/HA/HRPAuth-Yggdrasil-API/main.go:L192 @ 124b2fa1`
  registers `DELETE /texture/delete/:uuid/:type`; no
  `POST /texture/delete` exists (verified by route listing, 2026-11-04).
- HA-Contract legacy-passthrough statement (repo HA-Contract @ `21c7147d`):
  `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HASkinProxy/README.md`,
  section "Texture Management
  Passthrough (Legacy Compatibility)" — "Against the current HA upstream
  this path is not registered, so the forwarder receives a 404." Note: the
  contract text describes the `10426` target
  (`{upstream.base_url}/yggdrasil-api/texture/delete`); on `main` the
  target is `{upstream.base_url}/texture/delete` — both are unregistered.
- Relay prefix mechanics (repo HRPAuth @ `8dd67593`):
  [xref] `/home/lnb/Desktop/HA/HRPAuth/controllers/relay_controller.go:L172-L199 @ 8dd67593`
  (longest-prefix match),
  [xref] `/home/lnb/Desktop/HA/HRPAuth/controllers/relay_controller.go:L283-L316 @ 8dd67593`
  (forward with prefix stripped),
  [xref] `/home/lnb/Desktop/HA/HRPAuth/controllers/relay_controller.go:L274-L277 @ 8dd67593`
  (`/yggdrasil-api` allowed anonymously),
  [xref] `/home/lnb/Desktop/HA/HRPAuth/controllers/relay_controller.go:L286-L289 @ 8dd67593`
  (`/services/*` never relayed).
- HRPAuth root lacks Yggdrasil routes: HRPAuth `main.go` route listing —
  [xref] `/home/lnb/Desktop/HA/HRPAuth/main.go:L174 @ 8dd67593`
  (`api := r.Group("")`); no `sessionserver`,
  `/api/profiles/minecraft`, or `/textures` routes (verified 2026-11-04).
- Single BaseURL usage: every URL in `internal/hrpauth/client.go` is
  `c.BaseURL + path` (`internal/hrpauth/client.go:L51 @ b4a041ec`,
  `internal/hrpauth/client.go:L86 @ b4a041ec`,
  `internal/hrpauth/client.go:L147 @ b4a041ec`,
  `internal/hrpauth/client.go:L174 @ b4a041ec`,
  `internal/hrpauth/client.go:L203 @ b4a041ec`,
  `internal/hrpauth/client.go:L251 @ b4a041ec`,
  `internal/hrpauth/client.go:L312 @ b4a041ec`,
  `internal/hrpauth/client.go:L353 @ b4a041ec`); default base URL
  `config/config.go (field: base_url)`.
- Email check: `internal/hrpauth/client.go:L200-L227 @ b4a041ec`; consumer
  `internal/service/auth_service.go:L45-L52 @ b4a041ec`; upstream side
  [xref] `/home/lnb/Desktop/HA/HRPAuth/main.go:L198 @ 8dd67593`
  (`POST /user`; there is no `/api/user` route).
- Static bearer auth: `internal/hrpauth/client.go:L37-L47 @ b4a041ec`;
  introduced by commit `b19585ca417ce2e482ebe6fc7cb291ea9ea035aa`
  ("feat(upstream): implement client id/secret auth", 2026-10-04).
- Contract verdict on static bearer (repo HA-Contract @ `21c7147d`):
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L30 @ 21c7147d`
  (client_secret usage limited to `POST /oauth/token` client_credentials),
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L38 @ 21c7147d`
  and [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/tokens.md:L42 @ 21c7147d`
  (M-T is the only static bearer, marked legacy),
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`
  (client_credentials grant mandatory for microservice→core calls),
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L379-L382 @ 21c7147d`
  (auth-level resolution; unresolvable token → 0).
- `10426` fixes: branch tip `a83de4d259b66f0e160b8cfbc76f1801050ad07e`;
  commits `80fdcfb` (OAuth2 client auth + `/yggdrasil-api` paths),
  `6830c9c` (compile-time SDK package).
