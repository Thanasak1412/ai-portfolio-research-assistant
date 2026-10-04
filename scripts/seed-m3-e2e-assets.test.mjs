import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import {
  mkdtempSync,
  mkdirSync,
  writeFileSync,
  existsSync,
  rmSync,
} from "node:fs";
import { tmpdir } from "node:os";
import { resolve, join } from "node:path";
import test from "node:test";

const script = resolve("scripts/seed-m3-e2e-assets.sh");
const target =
  "postgres://portfolio:portfolio_test_local_only@postgres-test:5432/portfolio_test?sslmode=disable";

for (const [name, contents, override, accepted] of [
  [
    "exact disposable target",
    `COMPOSE_DATABASE_URL=${target}\n`,
    undefined,
    true,
  ],
  [
    "persistent development database",
    "COMPOSE_DATABASE_URL=postgres://portfolio@postgres:5432/portfolio\n",
    undefined,
    false,
  ],
  [
    "arbitrary remote URL",
    "COMPOSE_DATABASE_URL=postgres://example.invalid/app\n",
    undefined,
    false,
  ],
  ["missing target", "APP_ENV=test\n", undefined, false],
  [
    "duplicate target",
    `COMPOSE_DATABASE_URL=${target}\nCOMPOSE_DATABASE_URL=${target}\n`,
    undefined,
    false,
  ],
  [
    "override after allowed line",
    `COMPOSE_DATABASE_URL=${target}\nCOMPOSE_DATABASE_URL=other\n`,
    undefined,
    false,
  ],
  [
    "export override",
    `COMPOSE_DATABASE_URL=${target}\nexport COMPOSE_DATABASE_URL=other\n`,
    undefined,
    false,
  ],
  [
    "inherited conflicting URL",
    `COMPOSE_DATABASE_URL=${target}\n`,
    "other",
    false,
  ],
]) {
  test(`M3 seed safety: ${name}`, () => {
    const directory = mkdtempSync(join(tmpdir(), "m3-seed-safety-"));
    try {
      mkdirSync(join(directory, "apps/web/tests/m3-e2e/fixtures"), {
        recursive: true,
      });
      writeFileSync(
        join(directory, "apps/web/tests/m3-e2e/fixtures/assets.sql"),
        "SELECT 1;\n",
      );
      writeFileSync(join(directory, "test.env"), contents);
      // Stub only the CLI safety boundary; real SQL/HTTPS proof runs separately.
      writeFileSync(
        join(directory, "docker"),
        `#!/bin/sh
set -eu
test "$*" = 'compose --env-file test.env exec -T postgres-test psql -v ON_ERROR_STOP=1 -U portfolio -d portfolio_test'
touch invoked
`,
        { mode: 0o700 },
      );
      const env = { ...process.env, PATH: `${directory}:${process.env.PATH}` };
      delete env.COMPOSE_DATABASE_URL;
      if (override !== undefined) env.COMPOSE_DATABASE_URL = override;
      const result = spawnSync("sh", [script, "test.env"], {
        cwd: directory,
        env,
        encoding: "utf8",
      });
      assert.equal(result.status, accepted ? 0 : 1);
      assert.equal(existsSync(join(directory, "invoked")), accepted);
    } finally {
      rmSync(directory, { recursive: true, force: true });
    }
  });
}
