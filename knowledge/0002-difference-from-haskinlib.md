# 0002 — Difference from HASkinLib

## Status

Accepted

## Date

2026-11-04

## Context

HASkinProxy and HASkinLib have easily confused names, and both deal with
"skins". Operators and new contributors regularly ask which one does what,
and whether HASkinProxy fronts HASkinLib.

## Decision

The two services are disjoint in role and do not call each other:

| Aspect | HASkinProxy (this repo) | HASkinLib |
|---|---|---|
| Role | Yggdrasil → CSL protocol translation for MC clients | Texture storage, gallery, upload/delete management for users |
| State | In-memory freecache only | MySQL tables + on-disk texture/preview files |
| Auth | None of its own for the CSL surface (passthrough) | OAuth2 Bearer verified against HRPAuth |
| Upstream | Yggdrasil surface (via `upstream.base_url`) | HRPAuth core HTTP API |
| Clients | Minecraft clients with CustomSkinLoader | Web frontends / users; other services pulling by hash |

HASkinProxy's texture fetch goes to the **Yggdrasil surface**
(`GET {upstream.base_url}/textures/{hash}`), *not* to HASkinLib's
`GET /texture/pull/{hash}`. The bytes that the Yggdrasil surface serves are
the same files HASkinLib writes, but that handoff happens at the
filesystem/deployment level between HASkinLib and HRPAuth-Yggdrasil-API —
HASkinProxy is downstream of it and unaware of it.

## Consequences

- Debugging "skin missing in CSL client": check (in order) this proxy's
  cache TTL, upstream Yggdrasil surface availability, then the storage
  wiring between HASkinLib and HRPAuth-Yggdrasil-API (see HASkinLib
  knowledge base entry 0005).
- Constraint: do not add direct HASkinProxy → HASkinLib calls to "fix"
  upstream drift; the intended dependency is on the Yggdrasil surface only.
- Naming: HASkinProxy's internal Go module/package name is `haskinproxy`;
  the HRPAuth client package (`internal/hrpauth`) mirrors WinnerProxy's
  layout by deliberate convention, not shared code.

## Evidence

- Texture fetch from Yggdrasil surface:
  `internal/hrpauth/client.go::FetchTexture`
  (`{upstream.base_url}/yggdrasil-api/textures/{hash}`).
- No HASkinLib reference anywhere in this repo: search for `skinlib` in
  `*.go` returns no call site (verified 2026-11-04).
- No database dependency: `go.mod` has no SQL driver; only gin, yaml,
  freecache and oauth2 (plus testify for tests).
- HASkinLib side of the story (repo HASkinLib @ `54eee2f1`):
  `knowledge/0005-division-of-labor-with-skinproxy-and-yggdrasil.md`.
- Package layout convention ("mirroring ... WinnerProxy"):
  `internal/hrpauth/client.go:L1-L3`, `internal/router/router.go:L1-L2`.
