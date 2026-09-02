import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

import { parse } from "yaml";

const contractPath = new URL("../openapi/v1.yaml", import.meta.url);
const generatedPath = new URL("../generated/api.ts", import.meta.url);
const contract = parse(await readFile(contractPath, "utf8"));

const transactionOperations = {
  "/portfolios/{portfolioId}/transactions": ["post", "get"],
  "/portfolios/{portfolioId}/transactions/{transactionId}": ["get"],
  "/portfolios/{portfolioId}/transactions/{transactionId}/corrections": [
    "post",
  ],
};

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

function responseReference(path, method, status) {
  return operation(path, method).responses[String(status)]?.$ref;
}

function responseSchema(path, method, status) {
  const response = operation(path, method).responses[String(status)];
  const resolved = response.$ref
    ? contract.components.responses[referencedName(response.$ref)]
    : response;
  return resolveSchema(resolved.content?.["application/json"]?.schema);
}

function parameterReferences(path, method) {
  return operation(path, method).parameters.map((parameter) => parameter.$ref);
}

function assertBearer(path, method) {
  assert.deepEqual(operation(path, method).security, [{ BearerAuth: [] }]);
}

test("defines exactly the four approved immutable Transaction operations", () => {
  assert.deepEqual(
    Object.keys(contract.paths)
      .filter((path) => path.includes("/transactions"))
      .sort(),
    Object.keys(transactionOperations).sort(),
  );
  for (const [path, methods] of Object.entries(transactionOperations)) {
    for (const method of methods) operation(path, method);
  }

  assert.equal(
    contract.paths["/portfolios/{portfolioId}/transactions/{transactionId}"]
      .patch,
    undefined,
  );
  assert.equal(
    contract.paths["/portfolios/{portfolioId}/transactions/{transactionId}"]
      .delete,
    undefined,
  );
  assert.equal(
    contract.paths[
      "/portfolios/{portfolioId}/transactions/{transactionId}/reversal"
    ],
    undefined,
  );
  assert.equal(
    contract.paths[
      "/portfolios/{portfolioId}/transactions/{transactionId}/adjustment"
    ],
    undefined,
  );
  assert.equal(
    contract.components.schemas.AdjustmentTransactionCommand,
    undefined,
  );
});

test("requires BearerAuth everywhere and Idempotency-Key only for commands", () => {
  for (const [path, methods] of Object.entries(transactionOperations)) {
    for (const method of methods) assertBearer(path, method);
  }

  for (const [path, method] of [
    ["/portfolios/{portfolioId}/transactions", "post"],
    [
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
    ],
  ]) {
    assert.ok(
      parameterReferences(path, method).includes(
        "#/components/parameters/IdempotencyKey",
      ),
    );
  }
  for (const [path, method] of [
    ["/portfolios/{portfolioId}/transactions", "get"],
    ["/portfolios/{portfolioId}/transactions/{transactionId}", "get"],
  ]) {
    assert.equal(
      parameterReferences(path, method).includes(
        "#/components/parameters/IdempotencyKey",
      ),
      false,
    );
  }

  const key = contract.components.parameters.IdempotencyKey;
  assert.equal(key.required, true);
  assert.deepEqual(key.schema, {
    type: "string",
    minLength: 16,
    maxLength: 128,
    pattern: "^[A-Za-z0-9][A-Za-z0-9._~-]{15,127}$",
  });
  assert.match(key.description, /identical command/i);
  assert.match(key.description, /different command/i);
});

test("freezes public create kinds and the kind-discriminated field matrix", () => {
  assert.deepEqual(contract.components.schemas.TransactionCreateKind.enum, [
    "BUY",
    "SELL",
    "DIVIDEND",
    "DEPOSIT",
    "WITHDRAWAL",
    "FEE",
  ]);
  assert.deepEqual(contract.components.schemas.TransactionVisibleKind.enum, [
    "BUY",
    "SELL",
    "DIVIDEND",
    "DEPOSIT",
    "WITHDRAWAL",
    "FEE",
    "REVERSAL",
  ]);
  const commandMapping =
    contract.components.schemas.TransactionCommand.discriminator.mapping;
  assert.equal(Object.hasOwn(commandMapping, "REVERSAL"), false);
  assert.equal(Object.hasOwn(commandMapping, "ADJUSTMENT"), false);

  const expected = {
    BuyTransactionCommand: {
      required: [
        "kind",
        "assetId",
        "quantity",
        "unitPrice",
        "currency",
        "effectiveAt",
      ],
      properties: [
        "assetId",
        "currency",
        "effectiveAt",
        "externalReference",
        "fee",
        "kind",
        "note",
        "quantity",
        "unitPrice",
      ],
    },
    SellTransactionCommand: {
      required: [
        "kind",
        "assetId",
        "quantity",
        "unitPrice",
        "currency",
        "effectiveAt",
      ],
      properties: [
        "assetId",
        "currency",
        "effectiveAt",
        "externalReference",
        "fee",
        "kind",
        "note",
        "quantity",
        "unitPrice",
      ],
    },
    DividendTransactionCommand: {
      required: ["kind", "assetId", "amount", "currency", "effectiveAt"],
      properties: [
        "amount",
        "assetId",
        "currency",
        "effectiveAt",
        "externalReference",
        "kind",
        "note",
      ],
    },
    DepositTransactionCommand: {
      required: ["kind", "amount", "currency", "effectiveAt"],
      properties: [
        "amount",
        "currency",
        "effectiveAt",
        "externalReference",
        "kind",
        "note",
      ],
    },
    WithdrawalTransactionCommand: {
      required: ["kind", "amount", "currency", "effectiveAt"],
      properties: [
        "amount",
        "currency",
        "effectiveAt",
        "externalReference",
        "kind",
        "note",
      ],
    },
    FeeTransactionCommand: {
      required: ["kind", "amount", "currency", "effectiveAt"],
      properties: [
        "amount",
        "currency",
        "effectiveAt",
        "externalReference",
        "kind",
        "note",
      ],
    },
  };

  for (const [name, shape] of Object.entries(expected)) {
    const schema = contract.components.schemas[name];
    assert.equal(schema.additionalProperties, false, name);
    assert.deepEqual(schema.required, shape.required, name);
    assert.deepEqual(
      Object.keys(schema.properties).sort(),
      shape.properties,
      name,
    );
    assert.equal(
      schema.properties.effectiveAt.$ref,
      "#/components/schemas/TransactionEffectiveAt",
      name,
    );
    for (const forbidden of [
      "portfolioId",
      "ownerUserId",
      "userId",
      "gross",
      "net",
    ]) {
      assert.equal(
        Object.hasOwn(schema.properties, forbidden),
        false,
        `${name} ${forbidden}`,
      );
    }
  }
  assert.match(
    contract.components.schemas.BuyTransactionCommand.description,
    /amount is forbidden/i,
  );
  assert.match(
    contract.components.schemas.DividendTransactionCommand.description,
    /quantity, unitPrice, and fee are forbidden/i,
  );
  assert.match(
    contract.components.schemas.DepositTransactionCommand.description,
    /assetId, quantity, unitPrice, and fee are forbidden/i,
  );
});

test("uses USD decimal strings and distinct command and history timestamp rules", () => {
  for (const schemaName of [
    "DecimalString",
    "PositiveDecimalString",
    "NonNegativeDecimalString",
  ]) {
    const schema = contract.components.schemas[schemaName];
    assert.equal(schema.type, "string");
    assert.match(schema.pattern, /12/);
    assert.equal(schema.type === "number", false);
  }
  assert.equal(contract.components.schemas.TransactionCurrency.const, "USD");
  assert.match(
    contract.components.schemas.TransactionEffectiveAt.pattern,
    /\\\.\[0-9\]\{1,6\}.*Z\$/,
  );
  assert.match(
    contract.components.schemas.TransactionEffectiveAt.description,
    /future time/i,
  );
  assert.match(
    contract.components.schemas.TransactionEffectiveAt.description,
    /backdated time/i,
  );
  const historyTime = contract.components.schemas.TransactionHistoryTime;
  assert.match(historyTime.pattern, /\\\.\[0-9\]\{1,6\}.*Z\$/);
  assert.match(historyTime.description, /may be past or future/i);
  assert.match(historyTime.description, /no command-time or ledger-replay/i);
  for (const [schemaName, maxLength] of [
    ["TransactionNote", 2000],
    ["TransactionExternalReference", 256],
  ]) {
    const schema = contract.components.schemas[schemaName];
    assert.equal(schema.minLength, 0, schemaName);
    assert.equal(schema.maxLength, maxLength, schemaName);
    assert.match(schema.description, /including as an empty string/i);
    assert.match(schema.description, /preserved exactly as submitted/i);
    assert.match(
      schema.description,
      /without trimming or Unicode normalization/i,
    );
    assert.match(
      schema.description,
      /absent and empty are distinct semantic values/i,
    );
  }
  for (const [path, method] of [
    ["/portfolios/{portfolioId}/transactions", "post"],
    [
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
    ],
  ]) {
    assert.equal(operation(path, method).requestBody["x-max-bytes"], 8192);
  }
});

test("documents financial Asset eligibility without changing M2 catalog semantics", () => {
  const asset = contract.components.schemas.TransactionAssetId;
  assert.match(asset.description, /EQUITY or ETF/i);
  assert.match(asset.description, /NYSE, NASDAQ, NYSEARCA, or AMEX/i);
  assert.match(asset.description, /CRYPTO.*financially ineligible/i);
  assert.deepEqual(contract.components.schemas.AssetType.enum, [
    "EQUITY",
    "ETF",
    "CRYPTO",
  ]);
});

test("freezes deterministic history ordering, filters, and cursor grammar", () => {
  const list = operation("/portfolios/{portfolioId}/transactions", "get");
  assert.match(
    list.description,
    /effectiveAt descending, portfolioSequence descending, then transactionId descending/i,
  );
  assert.match(list.description, /from <= to/i);
  assert.match(list.description, /may be past or future instants/i);
  assert.match(
    list.description,
    /do not perform command-time or ledger-replay/i,
  );
  assert.deepEqual(
    parameterReferences("/portfolios/{portfolioId}/transactions", "get"),
    [
      "#/components/parameters/RequestCorrelationId",
      "#/components/parameters/PortfolioId",
      "#/components/parameters/TransactionKindFilter",
      "#/components/parameters/TransactionEffectiveAtFrom",
      "#/components/parameters/TransactionEffectiveAtTo",
      "#/components/parameters/TransactionIncludeReversals",
      "#/components/parameters/TransactionCursor",
      "#/components/parameters/TransactionPageLimit",
    ],
  );
  const cursor = contract.components.parameters.TransactionCursor.schema;
  assert.deepEqual(cursor, {
    type: "string",
    minLength: 4,
    maxLength: 512,
    pattern: "^v1\\.[A-Za-z0-9_-]{1,509}$",
  });
  assert.match(
    contract.components.parameters.TransactionCursor.description,
    /canonical UTF-8 JSON tuple/i,
  );
  assert.equal(
    contract.components.parameters.TransactionPageLimit.schema.default,
    50,
  );
  assert.equal(
    contract.components.parameters.TransactionPageLimit.schema.maximum,
    100,
  );
  assert.equal(
    contract.components.parameters.TransactionIncludeReversals.schema.default,
    true,
  );
  for (const parameterName of [
    "TransactionEffectiveAtFrom",
    "TransactionEffectiveAtTo",
  ]) {
    const parameter = contract.components.parameters[parameterName];
    assert.equal(
      parameter.schema.$ref,
      "#/components/schemas/TransactionHistoryTime",
    );
    assert.match(parameter.description, /may be past or future/i);
    assert.match(parameter.description, /no command-time or ledger-replay/i);
  }
});

test("freezes immutable public read records and correction chains", () => {
  const transaction = contract.components.schemas.Transaction;
  assert.deepEqual(Object.keys(transaction.properties).sort(), [
    "amount",
    "assetId",
    "correctionLinks",
    "createdAt",
    "currency",
    "effectiveAt",
    "externalReference",
    "fee",
    "id",
    "kind",
    "note",
    "portfolioSequence",
    "quantity",
    "unitPrice",
  ]);
  for (const forbidden of [
    "ownerUserId",
    "portfolioId",
    "idempotency",
    "audit",
    "outbox",
    "provider",
    "holding",
    "costBasis",
    "valuation",
  ]) {
    assert.equal(
      Object.hasOwn(transaction.properties, forbidden),
      false,
      forbidden,
    );
  }
  for (const financialField of ["quantity", "unitPrice", "fee", "amount"]) {
    const decimalVariant = transaction.properties[financialField].anyOf.find(
      (variant) => variant.$ref,
    );
    assert.equal(resolveSchema(decimalVariant).type, "string", financialField);
  }
  const links = contract.components.schemas.TransactionCorrectionLinks;
  assert.deepEqual(links.required, [
    "reversesTransactionId",
    "replacesTransactionId",
    "reversalTransactionId",
    "replacementTransactionId",
  ]);
  assert.match(links.description, /supports correction chains/i);

  const correction = responseSchema(
    "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
    "post",
    201,
  );
  assert.deepEqual(Object.keys(correction.properties).sort(), [
    "original",
    "replacement",
    "reversal",
  ]);
  assert.deepEqual(correction.required, [
    "original",
    "reversal",
    "replacement",
  ]);
  assert.match(
    operation(
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
    ).description,
    /existing direct reversal/i,
  );
  assert.match(
    operation(
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
    ).description,
    /internal REVERSAL target.*TRANSACTION_NOT_CORRECTABLE/i,
  );
  for (const field of ["note", "externalReference"]) {
    assert.match(
      transaction.properties[field].description,
      /null means absent/i,
    );
    assert.match(
      transaction.properties[field].description,
      /empty string means supplied empty/i,
    );
    assert.match(
      transaction.properties[field].description,
      /without trimming or Unicode normalization/i,
    );
  }
});

test("freezes ownership-safe, idempotency, and ledger command error mappings", () => {
  assert.equal(
    responseReference("/portfolios/{portfolioId}/transactions", "post", 404),
    "#/components/responses/PortfolioNotFound",
  );
  for (const [path, method] of [
    ["/portfolios/{portfolioId}/transactions/{transactionId}", "get"],
    [
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
    ],
  ]) {
    assert.equal(
      responseReference(path, method, 404),
      "#/components/responses/TransactionNotFound",
    );
  }
  assert.equal(
    responseReference("/portfolios/{portfolioId}/transactions", "post", 409),
    "#/components/responses/IdempotencyConflict",
  );
  assert.equal(
    responseReference(
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
      409,
    ),
    "#/components/responses/TransactionCorrectionConflict",
  );
  assert.equal(
    responseReference("/portfolios/{portfolioId}/transactions", "post", 400),
    "#/components/responses/InvalidTransactionRequest",
  );
  assert.equal(
    responseReference("/portfolios/{portfolioId}/transactions", "post", 422),
    "#/components/responses/TransactionCommandRejected",
  );
  assert.equal(
    responseReference(
      "/portfolios/{portfolioId}/transactions/{transactionId}/corrections",
      "post",
      422,
    ),
    "#/components/responses/TransactionCorrectionRejected",
  );
  for (const responseName of [
    "InvalidTransactionRequest",
    "TransactionCommandRejected",
    "IdempotencyConflict",
    "TransactionCorrectionConflict",
    "TransactionCorrectionRejected",
    "TransactionNotFound",
  ]) {
    const response = contract.components.responses[responseName];
    assert.equal(
      response.content["application/json"].schema.$ref,
      "#/components/schemas/ErrorEnvelope",
    );
    assert.ok(response.headers["X-Correlation-ID"]);
  }
  const codes = contract.components.schemas.ErrorCode.enum;
  for (const code of [
    "INVALID_IDEMPOTENCY_KEY",
    "INVALID_TRANSACTION_FIELDS",
    "INVALID_DECIMAL",
    "INVALID_EFFECTIVE_AT",
    "UNSUPPORTED_TRANSACTION_KIND",
    "ASSET_FINANCIALLY_INELIGIBLE",
    "UNSUPPORTED_TRANSACTION_CURRENCY",
    "INSUFFICIENT_ORDERED_ASSET_QUANTITY",
    "INVALID_BACKDATED_LEDGER",
    "IDEMPOTENCY_CONFLICT",
    "TRANSACTION_ALREADY_CORRECTED",
    "TRANSACTION_NOT_CORRECTABLE",
    "TRANSACTION_NOT_FOUND",
  ]) {
    assert.ok(codes.includes(code), code);
  }
  assert.match(
    contract.components.responses.TransactionCommandRejected.description,
    /ASSET_NOT_FOUND/i,
  );
  assert.doesNotMatch(
    contract.components.responses.TransactionCommandRejected.description,
    /UNSUPPORTED_TRANSACTION_KIND|UNSUPPORTED_TRANSACTION_CURRENCY/i,
  );
  assert.match(
    contract.components.responses.InvalidTransactionRequest.description,
    /UNSUPPORTED_TRANSACTION_KIND.*UNSUPPORTED_TRANSACTION_CURRENCY/i,
  );
  assert.match(
    contract.components.responses.TransactionCorrectionRejected.description,
    /TRANSACTION_NOT_CORRECTABLE.*internal\s+REVERSAL/i,
  );
  assert.match(
    contract.components.responses.PortfolioNotFound.description,
    /unrepresentable opaque route value/i,
  );
});

test("generates frozen Transaction types and introduces no M4 scope", async () => {
  const generated = await readFile(generatedPath, "utf8");
  for (const expected of [
    "TransactionCommand",
    "TransactionCorrectionCommand",
    "TransactionCorrectionResult",
    "TransactionListResponse",
    "TransactionVisibleKind",
  ]) {
    assert.match(generated, new RegExp(`\\b${expected}\\b`));
  }
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
