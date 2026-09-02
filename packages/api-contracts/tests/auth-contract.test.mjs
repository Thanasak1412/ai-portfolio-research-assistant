import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { parse } from "yaml";

const contractPath = new URL("../openapi/v1.yaml", import.meta.url);
const contract = parse(await readFile(contractPath, "utf8"));

const authOperations = {
  "/auth/register": "post",
  "/auth/login": "post",
  "/auth/refresh": "post",
  "/auth/logout": "post",
  "/auth/me": "get",
};

const m2Operations = {
  "/portfolios": ["post", "get"],
  "/portfolios/{portfolioId}": ["get", "patch"],
  "/portfolios/{portfolioId}/archive": ["post"],
  "/assets": ["get"],
  "/assets/{assetId}": ["get"],
};

const m3Operations = {
  "/portfolios/{portfolioId}/transactions": ["post", "get"],
  "/portfolios/{portfolioId}/transactions/{transactionId}": ["get"],
  "/portfolios/{portfolioId}/transactions/{transactionId}/corrections": [
    "post",
  ],
};

const expectedPaths = new Set([
  "/health/live",
  "/health/ready",
  ...Object.keys(authOperations),
  ...Object.keys(m2Operations),
  ...Object.keys(m3Operations),
]);

function operation(path, method) {
  const value = contract.paths?.[path]?.[method];
  assert.ok(value, `missing ${method.toUpperCase()} ${path}`);
  return value;
}

function referencedName(reference) {
  return reference?.split("/").at(-1);
}

function resolveSchema(schema) {
  if (schema?.$ref) {
    return contract.components.schemas[referencedName(schema.$ref)];
  }
  return schema;
}

function responseSchema(path, method, status) {
  const response = operation(path, method).responses[String(status)];
  const resolvedResponse = response.$ref
    ? contract.components.responses[referencedName(response.$ref)]
    : response;
  return resolveSchema(resolvedResponse.content?.["application/json"]?.schema);
}

function propertyNames(schema, found = new Set()) {
  const resolved = resolveSchema(schema);
  for (const name of Object.keys(resolved?.properties ?? {})) found.add(name);
  for (const member of resolved?.allOf ?? []) propertyNames(member, found);
  return found;
}

test("defines only the approved operational, Authentication, M2, and M3 operations", () => {
  assert.deepEqual(new Set(Object.keys(contract.paths)), expectedPaths);

  for (const [path, method] of Object.entries(authOperations))
    operation(path, method);
  for (const [path, methods] of Object.entries(m2Operations))
    for (const method of methods) operation(path, method);
  for (const [path, methods] of Object.entries(m3Operations))
    for (const method of methods) operation(path, method);

  const serverBase = contract.servers[0].url;
  for (const path of Object.keys(authOperations)) {
    assert.equal(`${new URL(serverBase).pathname}${path}`, `/api/v1${path}`);
  }
});

test("applies Bearer and refresh-cookie security only to their approved operations", () => {
  assert.deepEqual(contract.security, []);
  assert.equal(operation("/auth/register", "post").security, undefined);
  assert.equal(operation("/auth/login", "post").security, undefined);
  assert.deepEqual(operation("/auth/refresh", "post").security, [
    { RefreshCookieAuth: [] },
  ]);
  assert.deepEqual(operation("/auth/logout", "post").security, [
    { RefreshCookieAuth: [] },
  ]);
  assert.deepEqual(operation("/auth/me", "get").security, [{ BearerAuth: [] }]);

  assert.equal(contract.components.securitySchemes.BearerAuth.scheme, "bearer");
  assert.equal(
    contract.components.securitySchemes.BearerAuth.bearerFormat,
    "JWT",
  );
  assert.equal(
    contract.components.securitySchemes.RefreshCookieAuth.in,
    "cookie",
  );
  assert.equal(
    contract.components.securitySchemes.RefreshCookieAuth.name,
    "pra_rt_v1",
  );
});

test("keeps refresh credentials out of request and response bodies", () => {
  for (const path of ["/auth/refresh", "/auth/logout"]) {
    assert.equal(operation(path, "post").requestBody, undefined);
  }

  for (const path of ["/auth/register", "/auth/login"]) {
    const request = resolveSchema(
      operation(path, "post").requestBody.content["application/json"].schema,
    );
    assert.deepEqual(Object.keys(request.properties).sort(), [
      "email",
      "password",
    ]);

    const response = responseSchema(
      path,
      "post",
      path.endsWith("register") ? 201 : 200,
    );
    assert.equal(propertyNames(response).has("refreshToken"), false);
  }

  assert.equal(
    propertyNames(responseSchema("/auth/refresh", "post", 200)).has(
      "refreshToken",
    ),
    false,
  );
});

test("matches the approved credential input bounds", () => {
  const credentials = contract.components.schemas.CredentialsRequest;
  assert.deepEqual(credentials.required, ["email", "password"]);
  assert.equal(credentials.properties.email.format, "email");
  assert.equal(credentials.properties.password.minLength, 12);
  assert.equal(credentials.properties.password.maxLength, 1024);
  assert.equal(credentials.properties.password.writeOnly, true);
  assert.doesNotMatch(
    credentials.properties.password.description,
    /composition required/i,
  );
});

test("uses one generic public login failure", () => {
  const loginFailure = operation("/auth/login", "post").responses["401"];
  assert.equal(
    loginFailure.$ref,
    "#/components/responses/AuthenticationFailed",
  );
  assert.match(
    contract.components.responses.AuthenticationFailed.description,
    /AUTHENTICATION_FAILED/,
  );

  const responseNames = Object.keys(contract.components.responses).join(" ");
  assert.doesNotMatch(
    responseNames,
    /unknown.?email|disabled.?account|incorrect.?password/i,
  );

  const errorCodes = contract.components.schemas.ErrorCode.enum;
  assert.ok(errorCodes.includes("AUTHENTICATION_FAILED"));
  assert.equal(
    errorCodes.some((code) =>
      /UNKNOWN_EMAIL|DISABLED_ACCOUNT|INCORRECT_PASSWORD/.test(code),
    ),
    false,
  );
});

test("documents the exact cookie and browser-security requirements", () => {
  const cookieText = [
    contract.components.securitySchemes.RefreshCookieAuth.description,
    contract.components.headers.SetRefreshCookie.description,
    contract.components.headers.ClearRefreshCookie.description,
  ].join(" ");

  for (const expected of [
    "pra_rt_v1",
    "Secure",
    "HttpOnly",
    "SameSite=Lax",
    "/api/v1/auth",
    "host-only",
    "no Domain",
  ]) {
    assert.match(cookieText, new RegExp(expected.replace("/", "\\/"), "i"));
  }

  for (const path of ["/auth/refresh", "/auth/logout"]) {
    const parameterRefs = operation(path, "post").parameters.map(
      (item) => item.$ref,
    );
    assert.ok(parameterRefs.includes("#/components/parameters/RequiredOrigin"));
    assert.ok(parameterRefs.includes("#/components/parameters/RequestedWith"));
  }

  assert.equal(contract.components.parameters.RequiredOrigin.required, true);
  assert.equal(contract.components.parameters.RequestedWith.required, true);
  assert.equal(
    contract.components.parameters.RequestedWith.schema.const,
    "portfolio-web",
  );
});

test("returns only approved current-user fields", () => {
  const user = responseSchema("/auth/me", "get", 200);
  assert.deepEqual(Object.keys(user.properties).sort(), [
    "createdAt",
    "email",
    "id",
    "status",
    "updatedAt",
  ]);

  const serialized = JSON.stringify(user);
  for (const forbidden of [
    "password",
    "refresh",
    "tokenFamily",
    "audit",
    "rateLimit",
    "signingKey",
  ]) {
    assert.doesNotMatch(serialized, new RegExp(forbidden, "i"));
  }
});

test("uses the standard error envelope, correlation ID, and Retry-After", () => {
  const successStatuses = new Set(["200", "201", "204"]);
  for (const [path, method] of Object.entries(authOperations)) {
    for (const [status, response] of Object.entries(
      operation(path, method).responses,
    )) {
      const resolved = response.$ref
        ? contract.components.responses[referencedName(response.$ref)]
        : response;
      assert.ok(
        resolved.headers?.["X-Correlation-ID"],
        `${path} ${status} correlation`,
      );
      if (!successStatuses.has(status)) {
        assert.equal(
          resolved.content["application/json"].schema.$ref,
          "#/components/schemas/ErrorEnvelope",
        );
      }
    }
  }

  const rateLimited = contract.components.responses.RateLimited;
  assert.ok(rateLimited.headers["Retry-After"]);
  assert.match(rateLimited.description, /RATE_LIMIT_EXCEEDED/);

  for (const path of ["/auth/register", "/auth/login", "/auth/refresh"]) {
    assert.equal(
      operation(path, "post").responses["429"].$ref,
      "#/components/responses/RateLimited",
    );
  }
  assert.equal(operation("/auth/logout", "post").responses["429"], undefined);
});

test("introduces no scope beyond approved M3 Transactions", () => {
  for (const path of Object.keys(contract.paths)) {
    assert.doesNotMatch(
      path,
      /\/(holdings|prices|valuations|allocations|dashboard|alerts|documents|ai)(?:\/|$)/i,
    );
  }
  for (const schema of Object.keys(contract.components.schemas)) {
    assert.doesNotMatch(
      schema,
      /^(Holding|Price|Valuation|Allocation|Dashboard|Alert|Document|AI)/,
    );
  }
});
