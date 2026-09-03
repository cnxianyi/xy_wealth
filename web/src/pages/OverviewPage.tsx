import { useMemo } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Activity, Boxes, CircleDollarSign, RefreshCw, ShieldCheck, TrendingUp } from "lucide-react";
import { PageHeader } from "../components/layout/PageHeader";
import { Button } from "../components/ui/Button";
import { DataState } from "../components/ui/DataState";
import { DataTable } from "../components/ui/DataTable";
import { StatCard } from "../components/ui/StatCard";
import { StatusBadge } from "../components/ui/StatusBadge";
import { useSummary } from "../features/summary/useSummary";
import { flattenOverviewBalances } from "../features/summary/rows";
import { countBalances, countPositions, formatAmount, formatDate, productLabel, providerLabel } from "../lib/format";
import { ApiError } from "../lib/api-client";
import type { ExchangeData } from "../types/api";

function ProviderOverview({ exchange }: { exchange: ExchangeData }) {
  const products = exchange.products ?? [];
  const balanceCount = countBalances(exchange);
  const positionCount = countPositions(exchange);
  return (
    <article className="provider-card">
      <div className="provider-card-heading">
        <div className="provider-avatar">{providerLabel(exchange.provider).slice(0, 1)}</div>
        <div>
          <h3>{providerLabel(exchange.provider)}</h3>
          <p>{balanceCount} 个余额条目 · {positionCount} 个仓位</p>
        </div>
        <StatusBadge compact status={exchange.status} />
      </div>
      <div className="provider-products">
        <span className="provider-product-dot"><span />Spot</span>
        {products.map((product) => (
          <span className="provider-product-dot" key={product.product}>
            <span className={product.status !== "ok" ? "dot-warning" : undefined} />
            {productLabel(product.product)}
          </span>
        ))}
      </div>
      {exchange.error && <p className="inline-error">{exchange.error}</p>}
    </article>
  );
}

export function OverviewPage() {
  const { data, error, isError, isFetching, isLoading, refetch } = useSummary();
  const balanceRows = useMemo(() => (data ? flattenOverviewBalances(data) : []), [data]);
  const partialCount = data?.exchanges.filter((exchange) => exchange.status === "partial").length ?? 0;
  const activeCount = data?.exchanges.filter((exchange) => exchange.status === "ok" || exchange.status === "partial").length ?? 0;
  const providerCount = data?.exchanges.length ?? 0;
  const positionCount = data?.exchanges.reduce((count, exchange) => count + countPositions(exchange), 0) ?? 0;
  const balanceCount = data?.exchanges.reduce((count, exchange) => count + countBalances(exchange), 0) ?? 0;
  const productCount = data?.exchanges.reduce((count, exchange) => count + (exchange.products?.length ?? 0), 0) ?? 0;

  const columns = useMemo<ColumnDef<(typeof balanceRows)[number], unknown>[]>(
    () => [
      {
        accessorKey: "provider",
        header: "Provider",
        cell: (info) => <strong className="table-primary-text">{providerLabel(String(info.getValue()))}</strong>,
      },
      {
        accessorKey: "product",
        header: "产品",
        cell: (info) => productLabel(String(info.getValue())),
      },
      { accessorKey: "symbol", header: "资产" },
      {
        accessorKey: "total",
        header: "总量",
        cell: (info) => <span className="amount-emphasis">{formatAmount(info.getValue() as string | number)}</span>,
      },
      {
        accessorKey: "available",
        header: "可用",
        cell: (info) => formatAmount(info.getValue() as string | number),
      },
      {
        accessorKey: "locked",
        header: "冻结",
        cell: (info) => formatAmount(info.getValue() as string | number | undefined),
      },
    ],
    [],
  );

  return (
    <>
      <PageHeader
        eyebrow="ACCOUNT SNAPSHOT"
        title="概览"
        description="跨交易所的只读资产快照，保持对真实数据的忠实呈现。"
        action={
          <Button
            disabled={isFetching}
            icon={<RefreshCw className={isFetching ? "animate-spin" : undefined} size={16} />}
            onClick={() => void refetch()}
          >
            刷新数据
          </Button>
        }
      />

      {isLoading && <DataState state="loading" />}
      {isError && <DataState message={error instanceof ApiError ? error.message : "概览数据加载失败"} onRetry={() => void refetch()} state="error" />}
      {data && (
        <>
          <div className="snapshot-meta">
            <span><span className="live-pulse" />数据已同步</span>
            <span>更新于 {formatDate(data.generated_at)}</span>
          </div>
          {partialCount > 0 && (
            <div className="notice notice-warning" role="status">
              <Activity size={17} />
              <span>{partialCount} 个 Provider 仅部分成功，以下数据可能不完整。可在交易所页面查看具体错误。</span>
            </div>
          )}
          {providerCount === 0 && <DataState message="当前没有已配置的交易所 Provider" state="empty" />}
          {providerCount > 0 && (
            <>
              <div className="stats-grid">
                <StatCard detail={`${activeCount} / ${providerCount} 个正常响应`} icon={<ShieldCheck size={18} />} label="已连接 Provider" tone="mint" value={activeCount} />
                <StatCard detail="Spot 与合约账户合计" icon={<CircleDollarSign size={18} />} label="余额条目" tone="blue" value={balanceCount} />
                <StatCard detail="非零仓位" icon={<TrendingUp size={18} />} label="活跃仓位" tone="amber" value={positionCount} />
                <StatCard detail="已注册的账户产品" icon={<Boxes size={18} />} label="产品面" tone="ink" value={productCount} />
              </div>

              <section className="section-block">
                <div className="section-heading">
                  <div>
                    <p className="eyebrow">CONNECTED SOURCES</p>
                    <h2>Provider 状态</h2>
                  </div>
                  <span className="section-count">{providerCount} 个来源</span>
                </div>
                <div className="provider-grid">
                  {data.exchanges.map((exchange) => <ProviderOverview exchange={exchange} key={exchange.provider} />)}
                </div>
              </section>

              <section className="panel section-block">
                <div className="section-heading panel-heading">
                  <div>
                    <p className="eyebrow">BALANCE LEDGER</p>
                    <h2>余额速览</h2>
                  </div>
                  <span className="muted-label">不含零值</span>
                </div>
                {balanceRows.length ? (
                  <DataTable caption="余额速览" columns={columns} data={balanceRows.slice(0, 10)} emptyLabel="暂无余额" />
                ) : (
                  <DataState message="Provider 暂未返回余额数据" state="empty" />
                )}
                {balanceRows.length > 10 && <p className="table-footnote">已展示前 10 项，完整内容请前往持仓页面查看。</p>}
              </section>
            </>
          )}
          <p className="valuation-note"><CircleDollarSign size={15} />当前接口只提供原始资产数量，暂不计算统一法币估值与收益率。</p>
        </>
      )}
    </>
  );
}
