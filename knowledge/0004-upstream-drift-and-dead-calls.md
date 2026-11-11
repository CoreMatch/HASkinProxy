# 0004 — Upstream drift and dead calls

## Status

Accepted

## Context

The HA stack splits the Yggdrasil surface into the HRPAuth-Yggdrasil-API
microservice, reachable through HRPAuth's relay under the `/yggdrasil-api`
prefix (prefix stripped when forwarding), while the HRPAuth management
endpoints live at the HRPAuth root. This entry records how the proxy is
wired against both surfaces, and how the legacy delete passthrough is
mapped onto the route the current upstream actually registers.

## Decision

1. **The delete passthrough maps the legacy body onto the current
   upstream route.** `POST /{username}/texture/delete` parses the legacy
   JSON body (`remember_token, profile_id, texture_type, auth_type, uid,
   email`) and calls
   `DELETE {upstream.base_url}/yggdrasil-api/texture/delete/{uuid}/{type}`
   — the route HRPAuth-Yggdrasil-API actually registers — with
   `profile_id` as `{uuid}` and `texture_type` as `{type}`. The remaining
   legacy fields are ignored: the upstream identifies the account through
   the caller's Bearer token (forwarded unchanged) and validates profile
   ownership itself. The mapping exists because neither the legacy body
   shape nor a `POST /texture/delete` route exists on the current
   Yggdrasil-API (verified by route listing).
2. **Two bases from one `base_url`.** All outbound URLs derive from
   `upstream.base_url`: the Yggdrasil calls (`/api/profiles/minecraft`,
   `/sessionserver/session/minecraft/profile/{uuid}`, `/textures/{hash}`,
   `/texture/delete/{uuid}/{type}`) are built as `base_url + /yggdrasil-api + path` (the
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

- The delete passthrough keeps the legacy CSL-facing request shape (a
  POST with a JSON body) but reaches the current HA upstream; the
  upstream authenticates the caller and checks profile ownership, so the
  proxy stays out of authorization.
- `upstream.base_url` must point at the HRPAuth root (the default
  `http://localhost:2778` does): the relay serves `/yggdrasil-api/*` and
  skips `/services/*`, while the root serves the management endpoints —
  so both surfaces resolve from the one setting.

## Evidence

- Delete passthrough target and body mapping:
  `internal/hrpauth/client.go::DeleteTexture` (DELETE against
  `YggdrasilBaseURL` + `/texture/delete/{uuid}/{type}`, path segments
  escaped; `Authorization` header forwarded),
  `internal/handler/csl_handler.go::DeleteTexture` (legacy body parsed
  via `legacyDeleteRequest`; only `profile_id` and `texture_type`
  address the upstream).
- Upstream route reality (repo HRPAuth-Yggdrasil-API @ `124b2fa1`):
  [xref] `/home/lnb/Desktop/HA/HRPAuth-Yggdrasil-API/main.go:L192 @ 124b2fa1`
  registers `DELETE /texture/delete/:uuid/:type`; no
  `POST /texture/delete` exists (verified by route listing).
- HA-Contract legacy-passthrough statement (repo HA-Contract
  @ `21c7147d`): the contract's `haskinproxy-business.yaml` description
  still records the legacy mismatch ("the forwarder receives a 404");
  the proxy has since been migrated to the registered route (Decision 1),
  and this repo's `knowledge/0008-service-development-guide.md`
  ("Texture Management Passthrough") documents the current behavior.
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
