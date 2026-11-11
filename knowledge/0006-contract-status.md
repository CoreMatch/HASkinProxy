# 0006 — Contract status

## Status

Accepted

## Context

HA-Contract keeps per-service OpenAPI files under
`/home/lnb/Desktop/HA/HA-Contract/docs/api/openapi/`
(`hrpauth-business.yaml`, `haskinlib-business.yaml`,
`hrpauth-yggdrasil-business.yaml`, `matchcorearena-business.yaml`,
`cascade-backend-v1.yaml`, `haskinproxy-business.yaml`).
HASkinProxy's external surface is covered there by
`haskinproxy-business.yaml`.

## Decision

1. **Current state:** HASkinProxy's external surface is documented by the
   OpenAPI contract
   `/home/lnb/Desktop/HA/HA-Contract/docs/api/openapi/haskinproxy-business.yaml`
   (paths `/health`, `/{username}`, `/textures/{hash}`,
   `/{username}/texture/delete`), with the architecture flow and endpoint
   table in this repo's
   `/home/lnb/Desktop/HA/hrpauth/HASkinProxy/knowledge/0008-service-development-guide.md`
   and the CSL wire format in
   `/home/lnb/Desktop/HA/HA-Contract/docs/references/CustomSkinAPI.md`.
   The delete endpoint is documented as the legacy-shaped passthrough that
   maps `profile_id`/`texture_type` from the JSON body onto the upstream
   `DELETE /texture/delete/{uuid}/{type}` route — the same mapping the
   code implements (see [0004](./0004-upstream-drift-and-dead-calls.md)).
2. **Contract-first discipline** (policy):
   - External-surface changes are registered in
     `haskinproxy-business.yaml` first and pass the drift checks before
     the implementing code merges; this knowledge base never defines
     external behavior itself.
   - CI enforces the contract: `check-business-api-drift.sh` compares the
     routes registered in `internal/router/router.go` against the yaml,
     and `check-consumer-urls.sh` resolves the upstream URLs built in
     `internal/hrpauth/client.go` against the yggdrasil/hrpauth yamls.
   - When code and the contract disagree, the contract describes the
     intended external behavior and the code is fixed to match; this
     repo's README is deployment-oriented and kept in sync with the dev
     doc (the setup page is delivered as a compile-time SDK package).

## Consequences

- Drift findings about upstream calls belong in
  [0004](./0004-upstream-drift-and-dead-calls.md) here and are reflected
  contract-side in the OpenAPI file and the dev doc; keep them
  consistent.
- The OpenAPI file is machine-checked: a route registered without a
  contract entry (or a contract path with no route) fails the route drift
  gate, and an upstream URL that resolves to no contract path fails the
  consumer URL gate.

## Evidence

- Contract file and drift coverage: repo HA-Contract,
  `/home/lnb/Desktop/HA/HA-Contract/docs/api/openapi/haskinproxy-business.yaml`
  (including the legacy-shaped `POST /{username}/texture/delete` with the
  `profile_id`/`texture_type` request body), enforced by
  `/home/lnb/Desktop/HA/HA-Contract/scripts/check-business-api-drift.sh`
  (routes vs yaml) and
  `/home/lnb/Desktop/HA/HA-Contract/scripts/check-consumer-urls.sh`
  (upstream URLs).
- Dev doc as the external reference: repo HA-Contract,
  `/home/lnb/Desktop/HA/HASkinProxy/knowledge/0008-service-development-guide.md`
  (endpoint table and passthrough note);
  this repo's README links to it at `README.md:L13-L15 fp=d4ebdb5d ~ All interface documents and development specific`.
- README endpoint list: `README.md:L74-L79 fp=d82f0064 ~ Deployment facts`.
- `docs/api/README.md` positioning ("OpenAPI is the source for paths and
  response structures"): repo HA-Contract,
  `/home/lnb/Desktop/HA/HA-Contract/docs/api/README.md`.
