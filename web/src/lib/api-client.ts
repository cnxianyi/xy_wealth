import type { ErrorResponse, LoginResponse, SummarySnapshot } from "../types/api";
import { clearStoredToken, getStoredToken } from "./storage";

export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;

  constructor(status: number, message: string, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.body = body;
  }
}

export type UnauthorizedHandler = () => void;

export interface ApiClientOptions {
  baseUrl?: string;
  onUnauthorized?: UnauthorizedHandler;
  fetcher?: typeof fetch;
  getToken?: () => string | null;
}

function parseErrorMessage(body: unknown, fallback: string): string {
  if (!body || typeof body !== "object") return fallback;
  const payload = body as ErrorResponse;
  return payload.error?.message || payload.message || fallback;
}

/** Small fetch wrapper for the read-only API. It is intentionally framework-free. */
export class ApiClient {
  private readonly baseUrl: string;
  private readonly onUnauthorized?: UnauthorizedHandler;
  private readonly fetcher: typeof fetch;
  private readonly getToken: () => string | null;

  constructor(options: ApiClientOptions = {}) {
    this.baseUrl = options.baseUrl ?? "";
    this.onUnauthorized = options.onUnauthorized;
    this.fetcher = options.fetcher ?? fetch;
    this.getToken = options.getToken ?? getStoredToken;
  }

  async login(secret: string, signal?: AbortSignal): Promise<LoginResponse> {
    return this.request<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ secret }),
      signal,
      includeToken: false,
    });
  }

  async summary(signal?: AbortSignal): Promise<SummarySnapshot> {
    return this.request<SummarySnapshot>("/api/v1/summary?include_zero=false", { signal });
  }

  async session(signal?: AbortSignal): Promise<unknown> {
    return this.request<unknown>("/api/v1/auth/session", { signal });
  }

  async logout(signal?: AbortSignal): Promise<void> {
    await this.request<unknown>("/api/v1/auth/logout", { method: "POST", signal });
  }

  private async request<T>(
    path: string,
    options: RequestInit & { includeToken?: boolean } = {},
  ): Promise<T> {
    const { includeToken = true, ...requestOptions } = options;
    const headers = new Headers(requestOptions.headers);
    headers.set("Accept", "application/json");
    if (requestOptions.body && !headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
    if (includeToken) {
      const token = this.getToken();
      if (token) headers.set("x-token", token);
    }

    let response: Response;
    try {
      response = await this.fetcher(`${this.baseUrl}${path}`, {
        ...requestOptions,
        headers,
      });
    } catch (error) {
      if (error instanceof DOMException && error.name === "AbortError") throw error;
      throw new ApiError(0, "无法连接到服务，请检查后端是否已启动", error);
    }

    const body = await this.readBody(response);
    if (!response.ok) {
      if (response.status === 401) {
        clearStoredToken();
        this.onUnauthorized?.();
      }
      throw new ApiError(response.status, parseErrorMessage(body, `请求失败（${response.status}）`), body);
    }
    return body as T;
  }

  private async readBody(response: Response): Promise<unknown> {
    const contentType = response.headers.get("content-type") ?? "";
    if (response.status === 204) return undefined;
    if (contentType.includes("application/json")) {
      try {
        return await response.json();
      } catch {
        return undefined;
      }
    }
    const text = await response.text();
    if (!text) return undefined;
    try {
      return JSON.parse(text);
    } catch {
      return text;
    }
  }
}

export function createApiClient(onUnauthorized?: UnauthorizedHandler): ApiClient {
  return new ApiClient({ onUnauthorized });
}
