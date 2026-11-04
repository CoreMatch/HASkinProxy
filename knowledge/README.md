# HASkinProxy Knowledge Base

This directory is the project's own memo book: Architecture Decision
Records (ADRs) that explain **why the code looks the way it does**, kept for
self-hosting and maintenance.

## Positioning

- These notes serve the people who operate and maintain HASkinProxy. They
  record decisions, their costs, and the evidence behind them.
- **They do not define externally visible behavior.** External behavior is
  defined by [HA-Contract](https://github.com/CoreMatch/HA-Contract) —
  for this service primarily `docs/dev/HASkinProxy/README.md` (endpoint
  surface and passthrough semantics) and `docs/references/CustomSkinAPI.md`
  (the CSL protocol itself).
- If an entry here conflicts with HA-Contract, **fix the entry, not the
  contract**. If the code conflicts with the contract, record the drift in
  an entry and resolve it through HA-Contract's change process.

## Important baseline note

Entries were authored against branch `main` at `b4a041ec` and cite
`file:line` evidence from that tree. Note the repository situation
(verified 2026-11-04 with reproducible commands):

- **Branch `10426` exists — locally and on the remote.** The author
  questioned this name, so the verification is recorded here:
  - `git branch -a` (local clone): `* 10426`, `main`,
    `remotes/origin/10426`, `remotes/origin/main`.
  - `git ls-remote --heads origin` raw output:
    `1facc2d69d88218aac33417b0e717318ed717c61  refs/heads/10426` and
    `7dc747be62967c82ea0e106fc08b6ceece5d5336  refs/heads/main`.
  - `git ls-remote origin` shows `HEAD` pointing at the `10426` tip, i.e.
    the remote's default branch is `10426`.
  - Provenance: `git reflog show 10426` records
    `b19585c 10426@{2026-10-04 14:31:25 +0800}: branch: Created from
    b19585ca417ce2e482ebe6fc7cb291ea9ea035aa` — the branch was created in
    this local clone on 2026-10-04, before this knowledge base existed.
  - The string `10426` is **not** a commit-hash prefix
    (`git rev-parse --disambiguate=10426` returns nothing), appears in no
    tracked file other than this knowledge base, and in no commit message
    other than the knowledge-base merges. It does not appear in the other
    HA repositories either. The name was chosen by whoever created the
    branch locally (possibly a ticket/issue number — **not verified**).
- `main` (`b4a041ec`) and `10426` (tip `a83de4d2` before the knowledge
  merge) **diverged** from `b19585c` (`git merge-base main 10426`):
  `main` carries the password-reset auth flow; `10426` carries OAuth2
  client-credentials auth, `/yggdrasil-api` path prefixing, compile-time
  SDK package delivery, and the Docker build. Merging one into the other
  conflicts in business code. Where relevant, entries call out the
  divergence; see especially entry
  [0004](./0004-upstream-drift-and-dead-calls.md).
- The remote `main` branch had been absent and was recreated when this
  knowledge base was pushed (2026-11-04). Since `10426` is real and is the
  remote default, the recreated `main` is kept as the home of the
  main-baselined knowledge; deleting it again is the author's call.
- When reading an entry on a `10426` checkout, re-check the cited lines:
  `internal/hrpauth/client.go`, `internal/router/router.go` and `main.go`
  differ between the two branches.

Which branch is the long-term mainline (and whether entries should be
re-baselined onto `10426`) is **pending author confirmation**.

## Rules

- Entries are numbered `NNNN-<slug>.md`, written in English, and follow:
  Status / Date / Context / Decision / Consequences / Evidence.
- Evidence must be checked against the repository before writing
  (`file:line`, commit hash). Anything that could not be verified is
  marked `not verified`. Gaps are stated honestly, never papered over.
- New entries get the next free number; superseded entries change Status
  to `Superseded by NNNN` instead of being deleted.

## Entries

| # | Title | Status | Date | One-line assertion |
|---|-------|--------|------|--------------------|
| [0001](./0001-service-scope-csl-protocol-translation.md) | Service scope: CSL protocol translation | Accepted | 2026-08-14 | HASkinProxy is a stateless translator that turns Yggdrasil profile lookups into CustomSkinLoader JSON and proxies texture bytes; it is not a storage or auth service. |
| [0002](./0002-difference-from-haskinlib.md) | Difference from HASkinLib | Accepted | 2026-11-04 | HASkinProxy converts and caches; HASkinLib stores and manages. They share no database and HASkinProxy never calls HASkinLib — both hang off the Yggdrasil surface. |
| [0003](./0003-customskinloader-compatibility.md) | CustomSkinLoader compatibility | Accepted | 2026-08-14 | `GET /{username}.json` answers in CSL shape (`textures` map keyed by model/cape/elytra with hash values) and `GET /textures/{hash}` serves the bytes, both read-through cached. |
| [0004](./0004-upstream-drift-and-dead-calls.md) | Upstream drift and dead calls | Accepted | 2026-11-04 | On `main`, the delete passthrough 404s against the current upstream, and a single `base_url` cannot serve both the Yggdrasil surface (needs `/yggdrasil-api` relay prefix) and the management surface (needs the HRPAuth root). Branch `10426` partially fixes this but is unmerged. |
| [0005](./0005-webui-presence-and-sdk-integration.md) | WEBUI presence, relay and setup page | Accepted | 2026-09-01 | The proxy self-registers with HRPAuth (presence + relay rules) and serves a runtime SDK JS plus a `/customskinloader` setup page with an email-code password-reset helper flow. |
| [0006](./0006-contract-status.md) | Contract status | Accepted | 2026-11-04 | There is no OpenAPI contract file for HASkinProxy in HA-Contract; the external surface is documented in HA-Contract `docs/dev/HASkinProxy/README.md`, which already describes the newer (branch `10426`) design. |
