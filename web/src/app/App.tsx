import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { BrowserRouter, Navigate, Outlet, Route, Routes } from "react-router-dom";
import { AppShell } from "../components/layout/AppShell";
import { AuthProvider, useAuth } from "../features/auth/AuthContext";
import { ThemeProvider } from "../features/auth/ThemeContext";
import { ExchangesPage } from "../pages/ExchangesPage";
import { HoldingsPage } from "../pages/HoldingsPage";
import { LoginPage } from "../pages/LoginPage";
import { OverviewPage } from "../pages/OverviewPage";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnReconnect: true,
    },
  },
});

function RequireAuth() {
  const { isAuthenticated, isCheckingSession } = useAuth();
  if (isCheckingSession) {
    return <div className="route-loading"><span className="loading-dot" />正在验证会话…</div>;
  }
  return isAuthenticated ? <Outlet /> : <Navigate replace to="/login" />;
}

export function App() {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              <Route element={<LoginPage />} path="/login" />
              <Route element={<RequireAuth />}>
                <Route element={<AppShell />}>
                  <Route element={<OverviewPage />} index />
                  <Route element={<HoldingsPage />} path="holdings" />
                  <Route element={<ExchangesPage />} path="exchanges" />
                </Route>
              </Route>
              <Route element={<Navigate replace to="/" />} path="*" />
            </Routes>
          </BrowserRouter>
        </AuthProvider>
      </QueryClientProvider>
    </ThemeProvider>
  );
}
