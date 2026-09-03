import { useState, type FormEvent } from "react";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { ArrowRight, Eye, EyeOff, LockKeyhole, Orbit, ShieldCheck } from "lucide-react";
import { Button } from "../components/ui/Button";
import { useAuth } from "../features/auth/AuthContext";
import { ApiError } from "../lib/api-client";

export function LoginPage() {
  const { isAuthenticated, isCheckingSession, login } = useAuth();
  const navigate = useNavigate();
  const location = useLocation();
  const [secret, setSecret] = useState("");
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showSecret, setShowSecret] = useState(false);

  if (isAuthenticated && !isCheckingSession) return <Navigate replace to="/" />;

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const value = secret.trim();
    if (!value) {
      setError("请输入配置中的访问 Secret");
      return;
    }
    setError(null);
    setIsSubmitting(true);
    try {
      await login(value);
      const state = location.state as { from?: { pathname?: string } } | null;
      navigate(state?.from?.pathname || "/", { replace: true });
    } catch (requestError) {
      setError(requestError instanceof ApiError ? requestError.message : "登录失败，请稍后重试");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <main className="login-page">
      <div className="login-atmosphere login-atmosphere-one" />
      <div className="login-atmosphere login-atmosphere-two" />
      <section className="login-card" aria-labelledby="login-title">
        <div className="login-brand">
          <span className="brand-mark brand-mark-large"><Orbit size={22} strokeWidth={2.2} /></span>
          <span>XY Wealth</span>
        </div>
        <div className="login-heading">
          <p className="eyebrow">PRIVATE WORKSPACE</p>
          <h1 id="login-title">欢迎回来</h1>
          <p>输入本地配置中的 Secret，进入只读资产监控台。</p>
        </div>
        <form className="login-form" onSubmit={submit}>
          <label htmlFor="secret">访问 Secret</label>
          <div className="secret-input-wrap">
            <LockKeyhole aria-hidden="true" size={17} />
            <input
              autoComplete="current-password"
              autoFocus
              id="secret"
              onChange={(event) => setSecret(event.target.value)}
              placeholder="输入 Secret"
              type={showSecret ? "text" : "password"}
              value={secret}
            />
            <button
              aria-label={showSecret ? "隐藏 Secret" : "显示 Secret"}
              className="secret-toggle"
              onClick={() => setShowSecret((visible) => !visible)}
              type="button"
            >
              {showSecret ? <EyeOff aria-hidden="true" size={16} /> : <Eye aria-hidden="true" size={16} />}
            </button>
          </div>
          {error && <p className="form-error" role="alert">{error}</p>}
          <Button disabled={isSubmitting} size="lg" type="submit" variant="primary" icon={<ArrowRight size={17} />}>
            {isSubmitting ? "验证中…" : "进入工作台"}
          </Button>
        </form>
        <div className="login-security-note">
          <ShieldCheck size={16} />
          <span>Token 仅保存在当前浏览器会话，关闭窗口后自动清除。</span>
        </div>
      </section>
      <p className="login-footer">XY WEALTH / READ-ONLY ACCOUNT MONITOR</p>
    </main>
  );
}
