import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./api-client";
import { storeToken, TOKEN_STORAGE_KEY } from "./storage";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    headers: { "content-type": "application/json" },
    status,
  });
}

describe("ApiClient", () => {
  beforeEach(() => {
    window.sessionStorage.clear();
  });

  it("adds the configured session token to summary requests", async () => {
    storeToken("token-from-session");
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ generated_at: "now", exchanges: [], banks: [] }));
    const client = new ApiClient({ fetcher });

    await client.summary();

    expect(fetcher).toHaveBeenCalledWith(
      "/api/v1/summary?include_zero=false",
      expect.objectContaining({ headers: expect.any(Headers) }),
    );
    const request = fetcher.mock.calls[0]?.[1];
    expect(new Headers(request?.headers).get("x-token")).toBe("token-from-session");
  });

  it("calls the browser fetch function with the global receiver", async () => {
    const browserFetch = vi.fn(function (this: unknown) {
      if (this !== globalThis) throw new TypeError("Illegal invocation");
      return Promise.resolve(jsonResponse({ x_token: "new-token", expires_at: "tomorrow" }));
    }) as unknown as typeof fetch;
    vi.stubGlobal("fetch", browserFetch);
    const client = new ApiClient();

    await expect(client.login("configured-secret")).resolves.toMatchObject({ x_token: "new-token" });
    expect(browserFetch).toHaveBeenCalledOnce();
  });

  it("posts the secret without persisting it or attaching a stale token", async () => {
    storeToken("old-token");
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ x_token: "new-token", expires_at: "tomorrow" }));
    const client = new ApiClient({ fetcher });

    await client.login("configured-secret");

    expect(fetcher).toHaveBeenCalledWith(
      "/api/v1/auth/login",
      expect.objectContaining({
        body: JSON.stringify({ secret: "configured-secret" }),
        method: "POST",
      }),
    );
    const request = fetcher.mock.calls[0]?.[1];
    expect(new Headers(request?.headers).get("x-token")).toBeNull();
    expect(window.sessionStorage.getItem(TOKEN_STORAGE_KEY)).toBe("old-token");
  });

  it("clears the token and calls the unauthorized handler on 401", async () => {
    storeToken("expired-token");
    const onUnauthorized = vi.fn();
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(jsonResponse({ error: { message: "token expired" } }, 401));
    const client = new ApiClient({ fetcher, onUnauthorized });

    await expect(client.summary()).rejects.toMatchObject({ status: 401, message: "token expired" });
    expect(window.sessionStorage.getItem(TOKEN_STORAGE_KEY)).toBeNull();
    expect(onUnauthorized).toHaveBeenCalledOnce();
  });

  it("logs out through the authenticated endpoint", async () => {
    storeToken("active-token");
    const fetcher = vi.fn<typeof fetch>().mockResolvedValue(new Response(null, { status: 204 }));
    const client = new ApiClient({ fetcher });

    await client.logout();

    expect(fetcher).toHaveBeenCalledWith("/api/v1/auth/logout", expect.objectContaining({ method: "POST" }));
    const request = fetcher.mock.calls[0]?.[1];
    expect(new Headers(request?.headers).get("x-token")).toBe("active-token");
  });
});
