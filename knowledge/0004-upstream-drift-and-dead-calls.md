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
   With the default base URL (`http://localhost:2778`, the root), the CSL
   core flow 404s; with a `/yggdrasil-api` base URL, presence/relay
   registration and the email check 404 instead. There is no configuration
   on `main` that makes everything work.
3. **Email existence check uses a non-existent endpoint.**
   `GET {base}/api/user?email={email}` — HRPAuth root registers
   `POST /api/user` (no GET variant, no email query form). The reset-code
   flow treats any non-200 as "user not found", so the check degrades to
   always-false rather than erroring.
4. **Auth style is a static bearer — the contract does not allow it.**
   Service-side requests send `Authorization: Bearer {client_secret}` plus
   `X-Client-ID` instead of an OAuth2 client-credentials token exchange.
   Checked against the contract (2026-11-04):
   - HA-Contract `docs/dev/HRPAuth/tokens.md:30` — the "Built-in Super
     Client Secret" (`client_secret`) is a credential **for
     `client_credentials`**, and its listed usage endpoint is only
     `POST /oauth/token (grant_type=client_credentials)`.
   - HA-Contract `docs/dev/HRPAuth/microservices.md:94` — "Used for all
     microservice -> HRPAuth Core API calls. Microservices must provide a
     Bearer token obtained via the `client_credentials` grant."
   - HA-Contract `docs/dev/HRPAuth/microservices.md:382` — auth-level
     resolution: "no `Authorization: Bearer` → `0`; token equals M-T →
     `2`; resolvable OAuth2 token → `2` if service token, else `1`;
     unresolvable → `0`." A raw `client_secret` is not an issued token and
     is not the M-T, so it resolves to level 0.
   - The only static bearer the contract accepts is the legacy Maintenance
     Token (`tokens.md:38`, `microservices.md:380`), explicitly marked
     legacy (`tokens.md:42`).
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
  `internal/hrpauth/client.go:134-170` (URL at `:147`, JSON content type at
  `:153`), `internal/handler/csl_handler.go:138-194` (legacy field list at
  `:144-146`), HEAD of `main` `b4a041ec`.
- Upstream route reality (repo HRPAuth-Yggdrasil-API @ `124b2fa1`):
  `main.go:192` registers `DELETE /texture/delete/:uuid/:type`; no
  `POST /texture/delete` exists (verified by route listing, 2026-11-04).
- HA-Contract legacy-passthrough statement (repo HA-Contract @ `21c7147d`):
  `docs/dev/HASkinProxy/README.md`, section "Texture Management
  Passthrough (Legacy Compatibility)" — "Against the current HA upstream
  this path is not registered, so the forwarder receives a 404." Note: the
  contract text describes the `10426` target
  (`{upstream.base_url}/yggdrasil-api/texture/delete`); on `main` the
  target is `{upstream.base_url}/texture/delete` — both are unregistered.
- Relay prefix mechanics (repo HRPAuth @ `8dd67593`):
  `controllers/relay_controller.go:172-199` (longest-prefix match),
  `:283-316` (forward with prefix stripped), `:274-277` (`/yggdrasil-api`
  allowed anonymously), `:286-289` (`/services/*` never relayed).
- HRPAuth root lacks Yggdrasil routes: `HRPAuth/main.go` route listing —
  `api := r.Group("")` at `:174`; no `sessionserver`,
  `/api/profiles/minecraft`, or `/textures` routes (verified 2026-11-04).
- Single BaseURL usage: every URL in `internal/hrpauth/client.go` is
  `c.BaseURL + path` (`:51,86,147,174,203,251,312,353`); default base URL
  `config/config.go:101`.
- Email check: `internal/hrpauth/client.go:200-227`; consumer
  `internal/service/auth_service.go:45-52`; upstream side
  `HRPAuth/main.go:198` (`POST /api/user`).
- Static bearer auth: `internal/hrpauth/client.go:37-47`; introduced by
  commit `b19585ca417ce2e482ebe6fc7cb291ea9ea035aa` ("feat(upstream):
  implement client id/secret auth", 2026-10-04).
- Contract verdict on static bearer (repo HA-Contract @ `21c7147d`):
  `docs/dev/HRPAuth/tokens.md:30` (client_secret usage limited to
  `POST /oauth/token` client_credentials), `:38` and `:42` (M-T is the only
  static bearer, marked legacy), `docs/dev/HRPAuth/microservices.md:94`
  (client_credentials grant mandatory for microservice→core calls),
  `:379-382` (auth-level resolution; unresolvable token → 0).
- `10426` fixes: branch tip `a83de4d259b66f0e160b8cfbc76f1801050ad07e`;
  commits `80fdcfb` (OAuth2 client auth + `/yggdrasil-api` paths),
  `6830c9c` (compile-time SDK package).
