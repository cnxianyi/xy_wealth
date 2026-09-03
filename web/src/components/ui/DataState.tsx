import { AlertTriangle, Database, LoaderCircle, RefreshCw } from "lucide-react";
import { Button } from "./Button";

interface DataStateProps {
  state: "loading" | "error" | "empty";
  message?: string;
  onRetry?: () => void;
}

export function DataState({ state, message, onRetry }: DataStateProps) {
  if (state === "loading") {
    return (
      <div className="data-state" role="status" aria-label="加载中">
        <LoaderCircle className="animate-spin" size={24} />
        <p>正在同步账户数据…</p>
      </div>
    );
  }

  if (state === "error") {
    return (
      <div className="data-state data-state-error" role="alert">
        <AlertTriangle size={24} />
        <p>{message || "数据暂时不可用"}</p>
        {onRetry && <Button variant="secondary" size="sm" icon={<RefreshCw size={14} />} onClick={onRetry}>重试</Button>}
      </div>
    );
  }

  return (
    <div className="data-state" role="status">
      <Database size={24} />
      <p>{message || "暂无可展示的数据"}</p>
    </div>
  );
}
