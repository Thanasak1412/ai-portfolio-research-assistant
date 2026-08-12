import { afterEach, describe, expect, it, vi } from "vitest";

import {
  authenticationSessionLockName,
  createBrowserSessionCoordinator,
} from "@/features/auth/model/auth-browser-coordinator";

const originalBroadcastChannel = globalThis.BroadcastChannel;
const originalLocks = navigator.locks;

afterEach(() => {
  Object.defineProperty(navigator, "locks", {
    configurable: true,
    value: originalLocks,
  });
  globalThis.BroadcastChannel = originalBroadcastChannel;
});

describe("browser Authentication coordination", () => {
  it("uses one fixed non-sensitive Web Lock name", async () => {
    const request = vi.fn(
      async (_name: string, operation: () => Promise<string>) => operation(),
    );
    Object.defineProperty(navigator, "locks", {
      configurable: true,
      value: { request },
    });
    const coordinator = createBrowserSessionCoordinator();
    await expect(
      coordinator.withSessionTransition(async () => "complete"),
    ).resolves.toBe("complete");
    expect(request).toHaveBeenCalledWith(
      authenticationSessionLockName,
      expect.any(Function),
    );
    expect(authenticationSessionLockName).toBe("portfolio-auth-session-v1");
  });

  it("broadcasts only allowlisted state signals and ignores unrelated payloads", () => {
    const channels: FakeBroadcastChannel[] = [];
    class FakeBroadcastChannel {
      listener?: (event: MessageEvent<unknown>) => void;
      sent: unknown[] = [];
      constructor(readonly name: string) {
        channels.push(this);
      }
      addEventListener(
        _type: string,
        listener: (event: MessageEvent<unknown>) => void,
      ) {
        this.listener = listener;
      }
      removeEventListener() {
        this.listener = undefined;
      }
      postMessage(message: unknown) {
        this.sent.push(message);
      }
      close() {}
    }
    globalThis.BroadcastChannel =
      FakeBroadcastChannel as unknown as typeof BroadcastChannel;
    const coordinator = createBrowserSessionCoordinator();
    const listener = vi.fn();
    coordinator.subscribe(listener);
    coordinator.broadcast("session-invalidated");
    expect(channels[1]?.sent).toEqual([{ type: "session-invalidated" }]);
    expect(JSON.stringify(channels[1]?.sent)).not.toMatch(
      /accessToken|refreshToken|Authorization|Bearer|pra_rt_v1|@/,
    );
    channels[0]?.listener?.(
      new MessageEvent("message", { data: { type: "unexpected", token: "x" } }),
    );
    expect(listener).not.toHaveBeenCalled();
    channels[0]?.listener?.(
      new MessageEvent("message", { data: { type: "logout-complete" } }),
    );
    expect(listener).toHaveBeenCalledWith("logout-complete");
  });

  it("uses a local-only fallback when Web Locks are unavailable", async () => {
    Object.defineProperty(navigator, "locks", {
      configurable: true,
      value: undefined,
    });
    const operation = vi.fn().mockResolvedValue("complete");
    await expect(
      createBrowserSessionCoordinator().withSessionTransition(operation),
    ).resolves.toBe("complete");
    expect(operation).toHaveBeenCalledTimes(1);
  });
});
