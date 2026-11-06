# 0006 — Contract status

## Status

Accepted

## Context

HA-Contract keeps per-service OpenAPI files under
`/home/lnb/Desktop/HA/HA-Contract/docs/api/openapi/`
(`hrpauth-business.yaml`, `haskinlib-business.yaml`,
`hrpauth-yggdrasil-business.yaml`, `matchcorearena-business.yaml`,
`cascade-backend-v1.yaml`). HASkinProxy has **no** OpenAPI file there.

## Decision

1. **Current state:** HASkinProxy's external surface
   is documented in HA-Contract
   `/home/lnb/Desktop/HA/HASkinProxy/knowledge/0008-service-development-guide.md`
   (architecture flow, endpoint table, legacy delete-passthrough note) plus
   `/home/lnb/Desktop/HA/HA-Contract/docs/references/CustomSkinAPI.md`
   for the CSL wire format. There is no
   `haskinproxy-business.yaml`.
2. **Handling while the contract file is missing** (policy):
   - The implementation reference is the code plus the HA-Contract dev
     doc; this repo's README is deployment-oriented and currently in sync
     with the dev doc (the setup page is delivered as a compile-time SDK
     package).
   - Gaps are fixed **in HA-Contract** (add an OpenAPI file or extend the
     dev doc there). This knowledge base never defines external behavior to
     fill the gap.
   - When code and the contract dev doc disagree, the contract describes
     the intended external behavior and the code is fixed to match.

## Consequences

- Drift findings about dead calls belong in
  [0004](./0004-upstream-drift-and-dead-calls.md) here and are reflected
  contract-side in the dev doc; keep the two consistent.
- No OpenAPI means no machine-checkable path/response contract for this
  service; reviewers should compare against the dev doc manually.

## Evidence

- OpenAPI directory contents (no haskinproxy file): repo HA-Contract
  (@ `21c7147d`), `/home/lnb/Desktop/HA/HA-Contract/docs/api/openapi/`
  listing.
- Dev doc as the external reference: repo HA-Contract,
  `/home/lnb/Desktop/HA/HASkinProxy/knowledge/0008-service-development-guide.md`
  (endpoint table and passthrough note);
  this repo's README links to it at `README.md:L13-L15`.
- README endpoint list: `README.md:L74-L79`.
- `docs/api/README.md` positioning ("OpenAPI is the source for paths and
  response structures"): repo HA-Contract,
  `/home/lnb/Desktop/HA/HA-Contract/docs/api/README.md`.
