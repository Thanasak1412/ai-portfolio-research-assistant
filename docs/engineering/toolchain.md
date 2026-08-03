# Toolchain Compatibility Matrix

| Tool | Supported baseline | Pinning point |
|---|---|---|
| Go | 1.26.5 | `go.mod` and CI 1.26 patch line |
| Node.js | 24 LTS | `.node-version`, Docker major, CI major |
| pnpm | 10.18.3 | root `packageManager` |
| PostgreSQL | 17 | Docker Compose/CI images |
| Next.js | 16.2.11 Active LTS security release | web package manifest/lockfile |
| sqlc | 1.28.0 | CI install and documented local prerequisite |
| goose | 3.26.0 | Makefile, Dockerfile, and CI command |
| Playwright | locked by pnpm | lockfile and Chromium install step |

Dependency update PRs are created weekly. Security fixes can bypass the normal update window but must pass all quality gates. Major framework/tool changes require explicit technical-lead review and updated documentation.

