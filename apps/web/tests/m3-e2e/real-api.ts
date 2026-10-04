import { request as httpsRequest } from "node:https";

// Real HTTPS evidence only, outside Playwright's traced APIRequestContext:
// bearer headers stay in process memory, never in trace parameters/attachments.
// No response interception, cookie construction, application session API, or mock.
export async function realCommand(
  url: string,
  authorization: string,
  key: string,
  command: unknown,
): Promise<{ status: number; body: unknown }> {
  const target = new URL(url);
  const origin = new URL(
    process.env.M3_E2E_BASE_URL ?? "https://app.localhost:3443",
  );
  if (target.protocol !== "https:" || target.origin !== origin.origin)
    throw new Error("M3 API evidence must use the browser HTTPS origin.");
  const data = JSON.stringify(command);
  return new Promise((resolve, reject) => {
    const request = httpsRequest(
      target,
      {
        method: "POST",
        rejectUnauthorized:
          process.env.PLAYWRIGHT_AUTH_E2E_IGNORE_HTTPS_ERRORS !== "true",
        // Chromium resolves *.localhost to loopback; Node DNS does not always.
        ...(target.hostname === "app.localhost"
          ? { hostname: "127.0.0.1", servername: "app.localhost" }
          : {}),
        headers: {
          Host: target.host,
          Authorization: authorization,
          "Idempotency-Key": key,
          "Content-Type": "application/json",
          "Content-Length": Buffer.byteLength(data),
        },
        timeout: 10_000,
      },
      (response) => {
        const chunks: Buffer[] = [];
        response.on("data", (chunk: Buffer) => chunks.push(chunk));
        response.on("end", () => {
          try {
            resolve({
              status: response.statusCode ?? 0,
              body: JSON.parse(Buffer.concat(chunks).toString("utf8")),
            });
          } catch {
            reject(new Error("Real HTTPS command returned invalid JSON."));
          }
        });
        response.on("error", () =>
          reject(new Error("Real HTTPS response failed.")),
        );
      },
    );
    request.on("timeout", () => request.destroy());
    request.on("error", () =>
      reject(new Error("Real HTTPS command transport failed.")),
    );
    request.end(data);
  });
}
