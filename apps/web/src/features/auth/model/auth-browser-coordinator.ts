export type AuthenticationSignal =
  "session-established" | "session-invalidated" | "logout-complete";

export interface BrowserSessionCoordinator {
  withSessionTransition<T>(operation: () => Promise<T>): Promise<T>;
  broadcast(signal: AuthenticationSignal): void;
  subscribe(listener: (signal: AuthenticationSignal) => void): () => void;
}

export const authenticationSessionLockName = "portfolio-auth-session-v1";
const authenticationChannelName = "portfolio-auth-state-v1";
const allowedSignals = new Set<AuthenticationSignal>([
  "session-established",
  "session-invalidated",
  "logout-complete",
]);

export function createBrowserSessionCoordinator(): BrowserSessionCoordinator {
  return {
    async withSessionTransition<T>(operation: () => Promise<T>): Promise<T> {
      if (typeof navigator !== "undefined" && navigator.locks) {
        return navigator.locks.request(
          authenticationSessionLockName,
          operation,
        );
      }
      return operation();
    },
    broadcast(signal) {
      if (typeof BroadcastChannel === "undefined") return;
      const channel = new BroadcastChannel(authenticationChannelName);
      channel.postMessage({ type: signal });
      channel.close();
    },
    subscribe(listener) {
      if (typeof BroadcastChannel === "undefined") return () => undefined;
      const channel = new BroadcastChannel(authenticationChannelName);
      const receive = (event: MessageEvent<unknown>) => {
        if (!isAuthenticationSignalMessage(event.data)) return;
        listener(event.data.type);
      };
      channel.addEventListener("message", receive);
      return () => {
        channel.removeEventListener("message", receive);
        channel.close();
      };
    },
  };
}

function isAuthenticationSignalMessage(
  value: unknown,
): value is { type: AuthenticationSignal } {
  if (!value || typeof value !== "object" || !("type" in value)) return false;
  return (
    typeof value.type === "string" &&
    allowedSignals.has(value.type as AuthenticationSignal)
  );
}
