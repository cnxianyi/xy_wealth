import { Check, CircleAlert, CircleX, LoaderCircle } from "lucide-react";
import { cn } from "../../lib/cn";
import { statusLabel, statusTone } from "../../lib/format";

export function StatusBadge({ status, compact = false }: { status?: string; compact?: boolean }) {
  const tone = statusTone(status);
  const Icon = tone === "success" ? Check : tone === "warning" ? CircleAlert : tone === "danger" ? CircleX : LoaderCircle;
  return (
    <span className={cn("status-badge", `status-${tone}`, compact && "status-compact")}>
      <Icon size={compact ? 12 : 13} strokeWidth={2.2} />
      {statusLabel(status)}
    </span>
  );
}
