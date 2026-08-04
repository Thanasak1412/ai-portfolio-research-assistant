# ADR-013 — Solo Maintainer Merge Governance

**Status:** Accepted  
**Date:** 2026-08-04

## Context

The repository currently has one maintainer. A pull-request author cannot provide an independent approval for their own pull request, and there is no suitable collaborator available to do so. Requiring independent approval would therefore make normal, reviewed delivery impossible during the solo-maintainer phase.

## Decision

`main` remains protected and every change must use a pull request. The GitHub protection policy requires:

- zero approving reviews during the solo-maintainer phase;
- all seven mandatory checks—`frontend`, `backend`, `contracts-and-generation`, `database-integration`, `browser-e2e`, `compose-smoke`, and `secrets`—to pass without bypass;
- the branch to be up to date before merge;
- all review conversations to be resolved;
- force pushes and deletion of `main` to remain prohibited; and
- a completed self-review checklist in every pull request.

Independent-review and latest-push approval requirements are disabled only because they cannot be satisfied by a sole maintainer. If a genuine collaborator joins, a new governance ADR must reconsider and, if appropriate, restore independent approval.

## Consequences

This policy preserves an auditable pull-request record and mandatory automated quality/security gates without creating a fake reviewer or bypassing required status checks. The maintainer is responsible for performing and documenting the self-review before merge.
