# 0001 — Service scope: CSL protocol translation

## Status

Accepted

## Date

2026-08-14 (initial implementation, commit `8cc1cbb5`)

## Context

Minecraft clients using the CustomSkinLoader (CSL) mod do not speak
Yggdrasil; they expect a simple per-player JSON document
(`{username}.json`) plus a hash-addressed texture download URL. The HA stack
already exposes a Yggdrasil surface (HRPAuth + HRPAuth-Yggdrasil-API), so
CSL support is a pure protocol-translation problem.

## Decision

HASkinProxy is a **stateless protocol-conversion proxy** (Gin, default
listen `:2702`):

- `GET /{username}.json` — resolve username → UUID → Yggdrasil profile via
  the upstream, decode the base64 `textures` property, re-shape into CSL
  JSON (see [0003](./0003-customskinloader-compatibility.md)).
- `GET /textures/{hash}` — fetch raw texture bytes from the upstream and
  forward them with content headers.
- `POST /{username}/texture/delete` — legacy passthrough to an upstream
  delete endpoint; see
  [0004](./0004-upstream-drift-and-dead-calls.md) for its broken state.
- `GET /health` — liveness.
- On `main` only (runtime WEBUI surface): `GET /sdk/haskinproxy.js`,
  `GET /customskinloader`, and `/auth/*` helper endpoints for the setup
  page; see [0005](./0005-webui-presence-and-sdk-integration.md).

It keeps **no database**; the only state is an in-memory freecache
(profiles and texture bytes, TTL-based). Configuration is auto-generated
next to the executable on first run (same convention as WinnerProxy).

## Consequences

- Benefit: trivially horizontally scalable; restarts lose only cache warmth.
- Cost: cache is per-process; multiple replicas do not share entries.
- Constraint: anything about *stored* textures (upload, gallery, deletion
  semantics) belongs to HASkinLib; anything about *Yggdrasil protocol*
  belongs to HRPAuth-Yggdrasil-API. This service only re-shapes and caches.
- The external endpoint table is documented in HA-Contract
  (`docs/dev/HASkinProxy/README.md`); this entry records internal scope.

## Evidence

- Route registration: `internal/router/router.go:15-41` (HEAD of `main`,
  `b4a041ec`).
- Stateless design / freecache: `internal/cache/cache.go:14-22`;
  README claim "High Performance Caching ... freecache": `README.md:22`.
- Config auto-generation next to executable: `main.go:19-28,57-63`.
- Default listen `:2702`: `config/config.go:99`.
- Initial implementation commit: `8cc1cbb572294ad91719253fec74586f4a8c10b6`
  ("feat: implement HASkinProxy for Custom Skin API", 2026-08-14).
- Positioning statement: `README.md:3` ("converts Yggdrasil API logic into
  the CustomSkinLoader (CSL) compatible protocol format").
