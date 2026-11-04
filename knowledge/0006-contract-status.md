# 0006 — Contract status

## Status

Accepted

## Date

2026-11-04

## Context

HA-Contract keeps per-service OpenAPI files under `docs/api/openapi/`
(`hrpauth-business.yaml`, `haskinlib-business.yaml`,
`hrpauth-yggdrasil-business.yaml`, `matchcorearena-business.yaml`,
`cascade-backend-v1.yaml`). HASkinProxy has **no** OpenAPI file there.

## Decision

1. **Current state (verified 2026-11-04):** HASkinProxy's external surface
   is documented in HA-Contract `docs/dev/HASkinProxy/README.md`
   (architecture flow, endpoint table, legacy delete-passthrough note) plus
   `docs/references/CustomSkinAPI.md` for the CSL wire format. There is no
   `haskinproxy-business.yaml`.
2. **Handling while the contract file is missing** (policy):
   - The implementation reference is the code plus the HA-Contract dev doc;
     this repo's README is deployment-oriented and may lag (it still lists
     `GET /customskinloader`, which is accurate on `main` but superseded by
     the SDK-package design on branch `10426`).
   - Gaps are fixed **in HA-Contract** (add an OpenAPI file or extend the
     dev doc there). This knowledge base never defines external behavior to
     fill the gap.
   - When code and the contract dev doc disagree, the contract describes
     the intended external behavior; note that the dev doc currently
     describes the *branch `10426`* design (compile-time SDK, prefixed
     delete target) while `main` still implements the older mechanism —
     resolving that is the merge of `10426`, not a knowledge-base edit.

## Consequences

- Drift findings about dead calls belong in
  [0004](./0004-upstream-drift-and-dead-calls.md) here *and* are already
  reflected contract-side in the dev doc; keep the two consistent when the
  fix lands.
- No OpenAPI means no machine-checkable path/response contract for this
  service; reviewers should compare against the dev doc manually.

## Evidence

- OpenAPI directory contents (no haskinproxy file): repo HA-Contract
  (@ `21c7147d`), `docs/api/openapi/` listing (verified 2026-11-04).
- Dev doc as the external reference: repo HA-Contract,
  `docs/dev/HASkinProxy/README.md` (endpoint table and passthrough note);
  this repo's README links to it at `README.md:13-15`.
- README endpoint list including `GET /customskinloader`: `README.md:70-75`.
- `docs/api/README.md` positioning ("OpenAPI is the source for paths and
  response structures"): repo HA-Contract, `docs/api/README.md`.
