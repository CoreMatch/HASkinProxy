# 0004 — Upstream drift and dead calls

## Status

Accepted

## Context

The HA stack splits the Yggdrasil surface into the HRPAuth-Yggdrasil-API
microservice, reachable through HRPAuth's relay under the `/yggdrasil-api`
prefix (prefix stripped when forwarding), while the HRPAuth management
endpoints live at the HRPAuth root. This entry records how the proxy is
wired against both surfaces, and the one call that does not succeed
against the current upstream.

## Decision

1. **The delete passthrough is legacy-only (404 against the current HA
   upstream).** `POST /{username}/texture/delete` forwards the body and
   the caller's `Authorization` header unchanged to
   `POST {upstream.base_url}/yggdrasil-api/texture/delete`. The current
   HRPAuth-Yggdrasil-API registers no such route; it exposes
   `DELETE /texture/delete/{uuid}/{type}` instead, so the forwarded call
   receives a 404. The handler documents the legacy body fields
   (`remember_token, profile_id, texture_type, auth_type, uid, email`),
   which match neither the current Yggdrasil-API nor the current HASkinLib
   delete API (multipart `type`+`hash` with OAuth2 Bearer). HA-Contract
   documents this endpoint as legacy-only and states it 404s against the
   current upstream.
2. **Two bases from one `base_url`.** All outbound URLs derive from
   `upstream.base_url`: the Yggdrasil calls (`/api/profiles/minecraft`,
   `/sessionserver/session/minecraft/profile/{uuid}`, `/textures/{hash}`,
   `/texture/delete`) are built as `base_url + /yggdrasil-api + path` (the
   relay prefix), while the management calls (`/services/presence`,
   `/services/relay`, `/services/sdk-packages`, `/oauth/token`) are built
   against the HRPAuth root. The client therefore keeps `BaseURL` and
   `YggdrasilBaseURL` apart.
3. **Service-side auth is OAuth2 client-credentials.** When `client_id`
   and `client_secret` are configured, service-side requests carry a
   Bearer token obtained from `POST {base_url}/oauth/token`
   (`grant_type=client_credentials`); the contract mandates that grant
   for microservice → HRPAuth core calls.

## Consequences

- The delete passthrough is compatibility-only for legacy upstreams (per
  HA-Contract); do not build user-facing flows on it.
- `upstream.base_url` must point at the HRPAuth root (the default
  `http://localhost:2778` does): the relay serves `/yggdrasil-api/*` and
  skips `/services/*`, while the root serves the management endpoints —
  so both surfaces resolve from the one setting.

## Evidence

- Delete passthrough target and verbatim forwarding:
  `internal/hrpauth/client.go::DeleteTexture` (URL built from
  `YggdrasilBaseURL`; body and `Authorization` header forwarded),
  `internal/handler/csl_handler.go::DeleteTexture` (legacy body fields
  documented in the handler comment).
- Upstream route reality (repo HRPAuth-Yggdrasil-API @ `124b2fa1`):
  [xref] `/home/lnb/Desktop/HA/HRPAuth-Yggdrasil-API/main.go:L192 @ 124b2fa1`
  registers `DELETE /texture/delete/:uuid/:type`; no
  `POST /texture/delete` exists (verified by route listing).
- HA-Contract legacy-passthrough statement (repo HA-Contract
  @ `21c7147d`):
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HASkinProxy/README.md`,
  section "Texture Management
  Passthrough (Legacy Compatibility)" — "Against the current HA upstream
  this path is not registered, so the forwarder receives a 404."
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
  `/api/profiles/minecraft`, or `/textures` routes.
- Two bases: `internal/hrpauth/client.go::NewHAClient` (`YggdrasilBaseURL`
  = `BaseURL + "/yggdrasil-api"`), `internal/hrpauth/client.go::yggURL`;
  default base URL `config/config.go (field: base_url)`.
- OAuth2 client-credentials: `internal/hrpauth/client.go::NewHAClient`
  (`clientcredentials.Config` against `/oauth/token`),
  `internal/hrpauth/client.go::do` (token attached to outbound requests);
  contract mandate (repo HA-Contract @ `21c7147d`):
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`
  (microservice → core calls must use a Bearer token from the
  `client_credentials` grant).
