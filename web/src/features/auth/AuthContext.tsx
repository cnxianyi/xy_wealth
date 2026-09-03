import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { ApiClient, ApiError } from "../../lib/api-client";
import { clearStoredToken, getStoredToken, storeToken } from "../../lib/storage";

interface AuthContextValue {
  token: string | null;
  isAuthenticated: boolean;
  isCheckingSession: boolean;
  client: ApiClient;
  login: (secret: string) => Promise<void>;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() => getStoredToken());
  const [isCheckingSession, setIsCheckingSession] = useState(() => Boolean(getStoredToken()));

  const handleUnauthorized = useCallback(() => {
    clearStoredToken();
    setToken(null);
  }, []);

  const client = useMemo(
    () => new ApiClient({ onUnauthorized: handleUnauthorized, getToken: () => token }),
    [handleUnauthorized, token],
  );

  useEffect(() => {
    if (!token) {
      return;
    }
    let active = true;
    void client.session().catch(() => {
      // A transport error is surfaced by the page query; only a 401 clears auth.
    }).finally(() => {
      if (active) setIsCheckingSession(false);
    });
    return () => {
      active = false;
    };
  }, [client, token]);

  const login = useCallback(
    async (secret: string) => {
      const response = await client.login(secret);
      if (!response.x_token) {
        throw new ApiError(500, "登录响应缺少 x-token");
      }
      storeToken(response.x_token, response.expires_at);
      setToken(response.x_token);
    },
    [client],
  );

  const logout = useCallback(async () => {
    try {
      await client.logout();
    } catch {
      // Local logout must still complete if the server is unavailable.
    } finally {
      clearStoredToken();
      setToken(null);
    }
  }, [client]);

  const value = useMemo(
    () => ({ token, isAuthenticated: Boolean(token), isCheckingSession, client, login, logout }),
    [client, isCheckingSession, login, logout, token],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
