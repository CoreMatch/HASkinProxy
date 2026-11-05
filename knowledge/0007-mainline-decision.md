# 0007 — Mainline decision: `10426` is the main branch, `main` retired

## Status

Accepted (author ruling 2026-11-04: "如果分叉之后 10426 明显维护更新，则
直接将 10426 作为新的主分支，撤销 main；反之亦然。")

## Context

`main` and `10426` diverged from `b19585ca` (2026-10-04,
`git merge-base main 10426`). The knowledge base was initially
double-written to both branches. The author required a data-driven choice
of a single mainline, excluding this knowledge base's own commits from the
activity measurement.

## Decision

**`10426` is the main branch; `main` is retired** (archived by tag, then
deleted from the remote).

### Determination data (knowledge commits excluded)

Exclusion method: `git log --no-merges b19585c..<branch> -- . ':!knowledge'`
and the matching `rev-list --count` / `diff --stat` path exclusions.
Exclusion statistics: non-merge commits since the fork point are 5 (main)
vs 5 (10426) in total, of which 3 (main) vs 2 (10426) are knowledge-base
commits, leaving **2 vs 3 substantive commits**.

`main` substantive commits (both on 2026-10-04, none after):

- `4ae8a10` 2026-10-04 feat(auth): add password reset authentication system
- `b4a041e` 2026-10-04 feat(auth): add reset code verification flow
- `git diff --stat b19585c..main -- . ':!knowledge'`: 9 files,
  +478/−6 — the CSL-setup-page password-reset helper flow
  (`auth_handler.go`, `auth_service.go`, SMTP config, tests).

`10426` substantive commits:

- `6830c9c` 2026-10-09 refactor(webui): switch to compile-time SDK package
  delivery
- `80fdcfb` 2026-10-09 feat: add OAuth2 client auth and update API paths
- `a83de4d` 2026-11-03 feat(docker): add multi-stage container image build
- `git diff --stat b19585c..10426 -- . ':!knowledge'`: 17 files,
  +568/−292 — compile-time SDK package (`sdk/`, `sdk_upload.go`),
  OAuth2 client-credentials auth + `/yggdrasil-api` paths, Docker build.

Qualitative signals:

- Recency: last substantive commit `main` 2026-10-04 14:22 +0800 vs
  `10426` 2026-11-03 23:50 +0800 — a full month apart, `10426` still
  active the day before this decision.
- Contract alignment: HA-Contract
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L154 @ 21c7147d`
  removes the runtime JS SDK model that `main` still implements, and
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`
  mandates the client-credentials grant that only
  `10426` implements; HA-Contract's HASkinProxy dev doc describes the
  `10426` design.
- Deployment capability: only `10426` has the Docker build consumed by
  HA-Services-Deployer's compose setup.
- No cherry-pick backfills in either direction (`git cherry main 10426`
  and the reverse show no equivalent commits among the substantive ones).

Conclusion wording per the author's standard: `10426` is **明显 (clearly)**
more actively maintained; `main` is not.

## Consequences

- Remote default branch: `10426`.
- `main` is preserved at its tip by tag `archive/main-2026-11-04` and then
  deleted from the remote; local `main` and `refs/remotes/origin/main`
  removed. Reason for retirement: superseded by the clearly more active
  `10426`; its unique content (password-reset helper flow) was never
  merged into the mainline.
- `main`'s unique work is recoverable from the archive tag. Whether to
  cherry-pick the password-reset flow into the mainline is **pending author
  confirmation** (note: the contract removed the runtime WEBUI surface that
  flow belonged to, so a revival needs design work, not just a
  cherry-pick).
- The branch name `10426` was confirmed by the author: it
  denotes the date **2026-10-04** (`10/4/26`), matching the reflog branch
  creation time 2026-10-04 14:31 — a date marker, not a functional
  codename. The branch is still **not renamed**: the name is already
  referenced by existing clones, the remote default-branch setting, and
  deployment/CI expectations, and renaming would break them without any
  functional gain.
- Single mainline from now on: knowledge-base entries and fixes land only
  on `10426`; the previous double-write practice is ended.

## Evidence

- Fork point: `git merge-base main 10426` →
  `b19585ca417ce2e482ebe6fc7cb291ea9ea035aa`.
- Branch tips at decision time: `main` =
  `8a5f8f09570ea6672f8533a8b59e1f2d9b11da1d`, `10426` =
  `6ac25e3b00b663b3017cf0aa955ecb7c46c96e8b` (both including the
  knowledge-base corrections; substantive tips are `b4a041ec` and
  `a83de4d2` respectively).
- Archive tag: `archive/main-2026-11-04` → `8a5f8f09` (pushed to origin
  before the branch deletion).
- Contract citations: repo HA-Contract (@ `21c7147d`),
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L94 @ 21c7147d`,
  [xref] `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HRPAuth/microservices.md:L154 @ 21c7147d`;
  `/home/lnb/Desktop/HA/HA-Contract/docs/dev/HASkinProxy/README.md`
  (describes the `10426` design).
