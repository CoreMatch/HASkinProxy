# 0003 — CustomSkinLoader compatibility

## Status

Accepted

## Context

CustomSkinLoader clients request `{username}.json` from each configured API
root and expect a specific JSON shape; texture entries are hash-like
identifiers resolved against `{api}/textures/{hash}`. The proxy must map
Yggdrasil's signed-property model onto this flat shape.

## Decision

1. **Profile endpoint**: `GET /{username}.json`. Requests without the
   `.json` suffix get a bare 404 (the wildcard route serves only CSL
   lookups). Flow: cache check → upstream username→UUID
   (`POST /api/profiles/minecraft`) → upstream profile
   (`GET /sessionserver/session/minecraft/profile/{uuid}`) → decode the
   base64 `textures` property → emit CSL JSON.
2. **CSL shape**: `{"username": ..., "textures": {<key>: <hash>}}` where
   the key is the skin model (`default`, or the metadata `model` value such
   as `slim`) for `SKIN`, and `cape` / `elytra` for those types. The hash
   is the last path segment of the Yggdrasil texture URL.
3. **Texture endpoint**: `GET /textures/{hash}` — read-through cache;
   upstream bytes are forwarded with `Content-Type` (default `image/png`),
   `Last-Modified` and `Cache-Control` when present; cache hits add
   `X-Cache: HIT`.
4. **Caching**: profiles under `profile:{username}` with `cache.profile_ttl`
   (default 3600 s); texture bytes under `tex:{hash}` with
   `cache.texture_ttl` (default 86400 s). On a successful delete
   passthrough, the profile entry and the captured `tex:` entries are
   evicted (see [0004](./0004-upstream-drift-and-dead-calls.md)).

## Consequences

- Benefit: CSL clients need only point their API root at this proxy (or at
  the relayed `{main origin}/csl/`); no upstream changes required.
- Cost: the proxy trusts the upstream's texture URL shape (last segment =
  hash); a URL scheme change upstream silently breaks hash extraction.
- Cost: negative results are not cached; an unknown username costs two
  upstream calls on every request.
- Constraint: the CSL wire format itself is defined externally (HA-Contract
  `docs/references/CustomSkinAPI.md`); this service must follow it, not
  extend it ad hoc.

## Evidence

- Suffix gate and profile flow: `internal/handler/csl_handler.go::GetProfile`.
- CSL model struct: `internal/model/csl.go::CSLProfile`.
- Hash extraction from URL: `internal/handler/csl_handler.go::extractHash`.
- Texture forwarding + headers + `X-Cache`: `internal/handler/csl_handler.go::GetTexture`.
- Cache keys/TTLs: `internal/handler/csl_handler.go::GetProfile`
  (`profile:` keys), `internal/handler/csl_handler.go::GetTexture`
  (`tex:` keys); defaults `config/config.go (field: profile_ttl)`,
  `config/config.go (field: texture_ttl)`.
- Upstream calls used: `internal/hrpauth/client.go::GetUUIDByUsername`
  (`POST /api/profiles/minecraft`),
  `internal/hrpauth/client.go::GetProfileByUUID`
  (`GET /sessionserver/session/minecraft/profile/{uuid}`).
- CSL relay dest `/csl` registration (so clients can use the main service
  origin): `main.go::announcePresence`; introduced by commit
  `1755ae81c39a9c151dc53c770dc2beea72f887cf` ("feat(csl): add csl api relay
  and update webui config").
- External protocol reference: repo HA-Contract (@ `21c7147d`),
  `/home/lnb/Desktop/HA/HA-Contract/docs/references/CustomSkinAPI.md` and
  `/home/lnb/Desktop/HA/hrpauth/HASkinProxy/knowledge/0008-service-development-guide.md`
  (Architecture Flow section).
