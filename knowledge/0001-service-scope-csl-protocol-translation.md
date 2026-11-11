# 0001 — Service scope: CSL protocol translation

## Status

Accepted

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
  delete endpoint; legacy-only against the current HA upstream, see
  [0004](./0004-upstream-drift-and-dead-calls.md).
- `GET /health` — liveness.

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
  (`/home/lnb/Desktop/HA/hrpauth/HASkinProxy/knowledge/0008-service-development-guide.md`);
  this entry records internal scope.

## Evidence

- Route registration: `internal/router/router.go::New`.
- Stateless design / freecache: `internal/cache/cache.go::NewCache`;
  README claim "High Performance Caching ... freecache": `README.md:L22-L22 fp=5ce351f5 ~ - **High Performance Caching**: Uses `freecache``.
- Config auto-generation next to executable: `main.go::configPath`
  (path next to the executable), `config/config.go::LoadConfig`
  (generates a default file when missing).
- Default listen `:2702`: `config/config.go (field: listen_addr)`.
- Initial implementation commit: `8cc1cbb572294ad91719253fec74586f4a8c10b6`
  ("feat: implement HASkinProxy for Custom Skin API").
- Positioning statement: `README.md:L3-L3 fp=7204b3d4 ~ A lightweight proxy service that converts **Yggd` ("converts Yggdrasil API logic into
  the CustomSkinLoader (CSL) compatible protocol format").
