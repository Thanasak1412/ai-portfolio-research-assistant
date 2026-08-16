import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { parse } from "yaml";

const contractPath = new URL("../openapi/v1.yaml", import.meta.url);
const contract = parse(await readFile(contractPath, "utf8"));

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

function responseReference(path, method, status) {
  return operation(path, method).responses[String(status)]?.$ref;
}

function propertyNames(schema, found = new Set()) {
  const resolved = resolveSchema(schema);
  for (const name of Object.keys(resolved?.properties ?? {})) found.add(name);
  for (const member of resolved?.allOf ?? []) propertyNames(member, found);
  return found;
}

function assertBearer(path, method) {
  assert.deepEqual(operation(path, method).security, [{ BearerAuth: [] }]);
}

test("defines exactly the approved Portfolio operations and no generic delete", () => {
  assert.ok(operation("/portfolios", "post"));
  assert.ok(operation("/portfolios", "get"));
  assert.ok(operation("/portfolios/{portfolioId}", "get"));
  assert.ok(operation("/portfolios/{portfolioId}", "patch"));
  assert.ok(operation("/portfolios/{portfolioId}/archive", "post"));
  assert.equal(contract.paths["/portfolios/{portfolioId}"].delete, undefined);
  assert.equal(
    contract.paths["/portfolios/{portfolioId}/archive"].delete,
    undefined,
  );
});

test("freezes Portfolio public fields and immutable command boundaries", () => {
  const portfolio = contract.components.schemas.Portfolio;
  assert.deepEqual(Object.keys(portfolio.properties).sort(), [
    "archivedAt",
    "baseCurrency",
    "createdAt",
    "id",
    "name",
    "status",
    "updatedAt",
  ]);
  assert.deepEqual(contract.components.schemas.PortfolioStatus.enum, [
    "ACTIVE",
    "ARCHIVED",
  ]);
  assert.equal(contract.components.schemas.PortfolioBaseCurrency.const, "USD");

  const create = contract.components.schemas.CreatePortfolioRequest;
  assert.deepEqual(Object.keys(create.properties).sort(), [
    "baseCurrency",
    "name",
  ]);
  assert.deepEqual(create.required, ["name", "baseCurrency"]);

  const update = contract.components.schemas.UpdatePortfolioRequest;
  assert.deepEqual(Object.keys(update.properties), ["name"]);
  assert.equal(update.additionalProperties, false);
  assert.equal(update.minProperties, 1);

  const names = new Set([
    ...Object.keys(portfolio.properties),
    ...Object.keys(create.properties),
    ...Object.keys(update.properties),
  ]);
  for (const forbidden of [
    "ownerUserId",
    "owner_user_id",
    "userId",
    "normalizedName",
    "normalized_name",
    "aggregateVersion",
  ]) {
    assert.equal(names.has(forbidden), false);
  }
});

test("documents Portfolio lifecycle, list, archive, and ownership-safe errors", () => {
  const list = operation("/portfolios", "get");
  const status = list.parameters.find((parameter) =>
    parameter.$ref?.endsWith("/PortfolioStatusFilter"),
  );
  assert.ok(status);
  assert.equal(
    contract.components.parameters.PortfolioStatusFilter.schema.default,
    "ACTIVE",
  );
  assert.match(list.description, /updatedAt descending/i);
  assert.doesNotMatch(list.description, /page|cursor/i);

  const archive = operation("/portfolios/{portfolioId}/archive", "post");
  assert.ok(archive.responses["200"]);
  assert.equal(archive.responses["204"], undefined);
  assert.match(archive.description, /idempotent/i);
  assert.match(archive.description, /no restore/i);
  assert.deepEqual(
    propertyNames(
      responseSchema("/portfolios/{portfolioId}/archive", "post", 200),
    ),
    propertyNames(contract.components.schemas.Portfolio),
  );

  for (const [path, method, status] of [
    ["/portfolios/{portfolioId}", "get", "404"],
    ["/portfolios/{portfolioId}", "patch", "404"],
    ["/portfolios/{portfolioId}/archive", "post", "404"],
  ]) {
    assert.equal(
      responseReference(path, method, status),
      "#/components/responses/PortfolioNotFound",
    );
  }
  assert.match(
    contract.components.responses.PortfolioNotFound.description,
    /absent or is not owned/i,
  );
  assert.equal(
    responseReference("/portfolios", "post", "409"),
    "#/components/responses/PortfolioNameConflict",
  );
  assert.equal(
    responseReference("/portfolios/{portfolioId}", "patch", "409"),
    "#/components/responses/PortfolioArchived",
  );
});

test("requires Bearer authentication for every Portfolio operation", () => {
  for (const [path, method] of [
    ["/portfolios", "post"],
    ["/portfolios", "get"],
    ["/portfolios/{portfolioId}", "get"],
    ["/portfolios/{portfolioId}", "patch"],
    ["/portfolios/{portfolioId}/archive", "post"],
  ]) {
    assertBearer(path, method);
  }
});

test("defines only the approved read-only Asset operations", () => {
  assert.ok(operation("/assets", "get"));
  assert.ok(operation("/assets/{assetId}", "get"));
  assert.equal(contract.paths["/assets"].post, undefined);
  assert.equal(contract.paths["/assets/{assetId}"].patch, undefined);
  assert.equal(contract.paths["/assets/{assetId}"].delete, undefined);
});

test("freezes canonical Asset metadata and supported AssetType values", () => {
  const asset = contract.components.schemas.Asset;
  assert.deepEqual(Object.keys(asset.properties).sort(), [
    "assetType",
    "currency",
    "exchange",
    "id",
    "name",
    "symbol",
  ]);
  assert.deepEqual(contract.components.schemas.AssetType.enum, [
    "EQUITY",
    "ETF",
    "CRYPTO",
  ]);
  assert.equal(contract.components.schemas.AssetCurrency.const, "USD");

  const serialized = JSON.stringify(asset);
  for (const forbidden of [
    "ownerUserId",
    "portfolioId",
    "quantity",
    "averageCost",
    "costBasis",
    "price",
    "marketValue",
    "realizedPnl",
    "unrealizedPnl",
    "allocation",
    "providerPriceId",
  ]) {
    assert.doesNotMatch(serialized, new RegExp(forbidden, "i"));
  }
  assert.match(asset.properties.exchange.description, /CRYPTO/i);
  assert.match(asset.properties.exchange.description, /never a provider/i);
});

test("defines deterministic authenticated Asset discovery filters and pagination", () => {
  const list = operation("/assets", "get");
  const parameterNames = list.parameters.map((parameter) =>
    referencedName(parameter.$ref),
  );
  assert.deepEqual(parameterNames, [
    "RequestCorrelationId",
    "AssetSearch",
    "AssetTypeFilter",
    "AssetCursor",
    "AssetPageLimit",
  ]);
  assert.equal(
    contract.components.parameters.AssetPageLimit.schema.default,
    25,
  );
  assert.equal(
    contract.components.parameters.AssetPageLimit.schema.maximum,
    100,
  );
  assert.match(list.description, /case-insensitive/i);
  assert.match(list.description, /symbol, exchange, then id ascending/i);

  const page = contract.components.schemas.AssetListResponse;
  assert.deepEqual(Object.keys(page.properties).sort(), [
    "items",
    "nextCursor",
  ]);
  assert.deepEqual(page.required, ["items", "nextCursor"]);
  assertBearer("/assets", "get");
  assertBearer("/assets/{assetId}", "get");
  assert.equal(
    responseReference("/assets/{assetId}", "get", "404"),
    "#/components/responses/AssetNotFound",
  );
});

test("uses the existing standard error envelope and introduces no later M2 scope", () => {
  for (const response of [
    "PortfolioNameConflict",
    "PortfolioNotFound",
    "PortfolioArchived",
    "AssetNotFound",
  ]) {
    const value = contract.components.responses[response];
    assert.equal(
      value.content["application/json"].schema.$ref,
      "#/components/schemas/ErrorEnvelope",
    );
    assert.ok(value.headers["X-Correlation-ID"]);
  }

  for (const path of Object.keys(contract.paths)) {
    assert.doesNotMatch(
      path,
      /\/(transactions|holdings|prices|valuations|allocations|dashboard|alerts|documents|ai)(?:\/|$)/i,
    );
  }
  for (const schema of Object.keys(contract.components.schemas)) {
    assert.doesNotMatch(
      schema,
      /^(Transaction|Holding|Price|Valuation|Allocation|Dashboard|Alert|Document|AI)/,
    );
  }
});
