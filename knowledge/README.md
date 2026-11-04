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

## Mainline and baseline

**Single mainline (decided 2026-11-04):** `10426` is the main branch;
`main` has been retired (archived as tag `archive/main-2026-11-04`, then
deleted from the remote). From now on, knowledge-base entries and fixes land
**only on `10426`** — the earlier double-write practice is ended. The full
decision record with data and commands is entry
[0007](./0007-mainline-decision.md).

Baseline facts for readers:

- Entries were originally authored against `main` @ `b4a041ec` and cite
  `file:line` evidence from that tree. Where a citation touches
  `internal/hrpauth/client.go`, `internal/router/router.go` or `main.go`,
  re-check against the `10426` tree — these files differ between the two
  branches. Entry [0004](./0004-upstream-drift-and-dead-calls.md) records
  which drift items `10426` fixes.
- Branch name note: the author confirmed on 2026-11-05 that `10426`
  denotes the date **2026-10-04** (`10/4/26`) — it matches the reflog
  branch-creation time 2026-10-04 14:31. The name is a date marker, not a
  functional codename. The branch is **not renamed**: existing clones, the
  remote default-branch setting, and deployment/CI expectations already
  reference this name, and renaming would break them without functional
  gain.
- The retired `main`'s unique work (the CSL-setup-page password-reset
  helper flow, `4ae8a10` + `b4a041e`) is preserved at tag
  `archive/main-2026-11-04`; reviving it is pending author confirmation.

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
| [0004](./0004-upstream-drift-and-dead-calls.md) | Upstream drift and dead calls | Accepted | 2026-11-04 | On the retired `main`, the delete passthrough 404s against the current upstream, and a single `base_url` cannot serve both the Yggdrasil surface (needs `/yggdrasil-api` relay prefix) and the management surface (needs the HRPAuth root). Branch `10426` — the mainline since the 2026-11-04 decision (entry 0007) — fixes these items; the entry records the drift history. |
| [0005](./0005-webui-presence-and-sdk-integration.md) | WEBUI presence, relay and setup page | Accepted | 2026-09-01 | The proxy self-registers with HRPAuth (presence + relay rules). On the retired `main` it also served a runtime SDK JS plus a `/customskinloader` setup page with an email-code password-reset helper flow; the `10426` mainline replaces that with a compile-time SDK package. |
| [0006](./0006-contract-status.md) | Contract status | Accepted | 2026-11-04 | There is no OpenAPI contract file for HASkinProxy in HA-Contract; the external surface is documented in HA-Contract `docs/dev/HASkinProxy/README.md`, which already describes the newer (branch `10426`) design. |
| [0007](./0007-mainline-decision.md) | Mainline decision: `10426` wins, `main` retired | Accepted | 2026-11-04 | Excluding knowledge commits, `10426` has 3 substantive commits (last 2026-11-03, contract-aligned, Docker-capable) vs `main`'s 2 (last 2026-10-04) — clearly more active; `main` archived as `archive/main-2026-11-04` and deleted. |
