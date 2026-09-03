import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { App } from "../app/App";

const snapshot = {
  generated_at: "2026-09-03T09:00:00Z",
  exchanges: [
    {
      provider: "binance",
      status: "ok",
      balances: [{ symbol: "BTC", free: "1.2", locked: "0", total: "1.2" }],
      products: [],
    },
  ],
  banks: [],
};

describe("App authentication flow", () => {
  it("logs in with the configured secret and renders the overview", async () => {
    window.history.pushState({}, "", "/login");
    const fetchMock = vi.fn<typeof fetch>((input) => {
      const url = String(input);
      if (url.endsWith("/auth/login")) {
        return Promise.resolve(new Response(JSON.stringify({ x_token: "session-token", expires_at: "tomorrow" }), { headers: { "content-type": "application/json" } }));
      }
      if (url.endsWith("/auth/session")) {
        return Promise.resolve(new Response(JSON.stringify({ authenticated: true }), { headers: { "content-type": "application/json" } }));
      }
      return Promise.resolve(new Response(JSON.stringify(snapshot), { headers: { "content-type": "application/json" } }));
    });
    vi.stubGlobal("fetch", fetchMock);

    render(<App />);
    fireEvent.change(screen.getByLabelText("访问 Secret"), { target: { value: "configured-secret" } });
    fireEvent.click(screen.getByRole("button", { name: "进入工作台" }));

    expect(await screen.findByRole("heading", { name: "概览" })).toBeInTheDocument();
    expect((await screen.findAllByText("Binance")).length).toBeGreaterThan(0);
    await waitFor(() => expect(window.sessionStorage.getItem("xy-wealth.x-token")).toBe("session-token"));
    expect(fetchMock.mock.calls.some(([input]) => String(input).endsWith("/summary?include_zero=false"))).toBe(true);
  });

  it("returns to login when a stored token is no longer valid", async () => {
    window.history.pushState({}, "", "/");
    window.sessionStorage.setItem("xy-wealth.x-token", "expired-token");
    vi.stubGlobal(
      "fetch",
      vi.fn<typeof fetch>().mockResolvedValue(
        new Response(JSON.stringify({ error: { code: "unauthorized", message: "expired" } }), {
          headers: { "content-type": "application/json" },
          status: 401,
        }),
      ),
    );

    render(<App />);

    expect(await screen.findByRole("heading", { name: "欢迎回来" })).toBeInTheDocument();
    expect(window.sessionStorage.getItem("xy-wealth.x-token")).toBeNull();
  });
});
