import { useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Activity, ChartNoAxesCombined, CircleHelp, LogOut, Menu, Moon, Orbit, PanelLeft, Sun, WalletCards, X } from "lucide-react";
import { cn } from "../../lib/cn";
import { useAuth } from "../../features/auth/AuthContext";
import { useTheme } from "../../features/auth/ThemeContext";

const navItems = [
  { to: "/", label: "概览", description: "账户全景", icon: ChartNoAxesCombined, end: true },
  { to: "/holdings", label: "持仓", description: "余额与仓位", icon: WalletCards },
  { to: "/exchanges", label: "交易所", description: "连接状态", icon: Activity },
];

function Navigation({ mobile = false, onNavigate }: { mobile?: boolean; onNavigate?: () => void }) {
  return (
    <nav className={cn(mobile ? "mobile-nav-links" : "sidebar-nav")} aria-label="主导航">
      {navItems.map(({ to, label, description, icon: Icon, end }) => (
        <NavLink
          className={({ isActive }) => cn("nav-link", isActive && "nav-link-active")}
          end={end}
          key={to}
          onClick={onNavigate}
          to={to}
        >
          <Icon size={mobile ? 20 : 18} strokeWidth={1.8} />
          <span className={mobile ? "mobile-nav-label" : undefined}>
            <span>{label}</span>
            {!mobile && <small>{description}</small>}
          </span>
        </NavLink>
      ))}
    </nav>
  );
}

export function AppShell() {
  const { logout } = useAuth();
  const { theme, toggleTheme } = useTheme();
  const navigate = useNavigate();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  const handleLogout = () => {
    logout();
    navigate("/login", { replace: true });
  };

  return (
    <div className={cn("app-shell", sidebarCollapsed && "sidebar-is-collapsed")}>
      <aside className="sidebar">
        <div className="brand-lockup">
          <span className="brand-mark"><Orbit size={19} strokeWidth={2.2} /></span>
          <span>
            <strong>XY Wealth</strong>
            <small>READ-ONLY DESK</small>
          </span>
        </div>
        <Navigation />
        <div className="sidebar-footer">
          <div className="sidebar-note">
            <CircleHelp size={16} />
            <span>只读账户监控</span>
          </div>
          <button
            aria-label={sidebarCollapsed ? "展开侧栏" : "收起侧栏"}
            className="sidebar-collapse"
            onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
            title={sidebarCollapsed ? "展开侧栏" : "收起侧栏"}
            type="button"
          >
            <PanelLeft size={16} />
            <span>Workspace</span>
          </button>
        </div>
      </aside>

      <div className="app-main">
        <header className="topbar">
          <div className="mobile-brand">
            <span className="brand-mark"><Orbit size={17} strokeWidth={2.2} /></span>
            <strong>XY Wealth</strong>
          </div>
          <div className="topbar-actions">
            <button
              aria-label={theme === "dark" ? "切换到浅色主题" : "切换到深色主题"}
              className="icon-button"
              onClick={toggleTheme}
              title={theme === "dark" ? "浅色主题" : "深色主题"}
              type="button"
            >
              {theme === "dark" ? <Sun size={18} /> : <Moon size={18} />}
            </button>
            <button className="logout-button" onClick={handleLogout} type="button">
              <LogOut size={16} />
              <span>退出</span>
            </button>
            <button
              aria-expanded={mobileMenuOpen}
              aria-label={mobileMenuOpen ? "关闭菜单" : "打开菜单"}
              className="mobile-menu-button icon-button"
              onClick={() => setMobileMenuOpen((open) => !open)}
              type="button"
            >
              {mobileMenuOpen ? <X size={20} /> : <Menu size={20} />}
            </button>
          </div>
        </header>
        {mobileMenuOpen && (
          <div className="mobile-menu">
            <Navigation mobile onNavigate={() => setMobileMenuOpen(false)} />
          </div>
        )}
        <main className="content"><Outlet /></main>
        <nav className="mobile-bottom-nav" aria-label="移动端主导航">
          <Navigation mobile />
        </nav>
      </div>
    </div>
  );
}
