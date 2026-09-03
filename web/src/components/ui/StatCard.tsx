import type { ReactNode } from "react";
import { cn } from "../../lib/cn";

interface StatCardProps {
  label: string;
  value: string | number;
  detail?: string;
  icon: ReactNode;
  tone?: "mint" | "blue" | "amber" | "ink";
}

export function StatCard({ label, value, detail, icon, tone = "mint" }: StatCardProps) {
  return (
    <article className={cn("stat-card", `stat-${tone}`)}>
      <div className="stat-card-top">
        <span className="stat-label">{label}</span>
        <span className="stat-icon">{icon}</span>
      </div>
      <strong className="stat-value">{value}</strong>
      {detail && <span className="stat-detail">{detail}</span>}
    </article>
  );
}
