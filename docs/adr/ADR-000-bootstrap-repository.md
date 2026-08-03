# ADR-000 — Bootstrap Repository Boundary

**Status:** Accepted.

The product is isolated in `ai-portfolio-research-assistant/` so the unrelated LINE application in the parent workspace remains unchanged. The approved dedicated repository is `github.com/Thanasak1412/ai-portfolio-research-assistant`, public, with `main` as its default branch. The Go module path matches that repository identity.

This decision prevents unrelated work, CI, secrets, and deployment history from being combined. It does not authorize business-feature implementation.
