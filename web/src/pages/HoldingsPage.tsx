import { useMemo, useState } from "react";
import type { ColumnDef } from "@tanstack/react-table";
import { Filter, RefreshCw, Search, SlidersHorizontal, WalletCards } from "lucide-react";
import { PageHeader } from "../components/layout/PageHeader";
import { Button } from "../components/ui/Button";
import { DataState } from "../components/ui/DataState";
import { DataTable } from "../components/ui/DataTable";
import { StatusBadge } from "../components/ui/StatusBadge";
import { useSummary } from "../features/summary/useSummary";
import { flattenHoldings, type HoldingKind, type HoldingRow } from "../features/summary/rows";
import { formatAmount, productLabel, providerLabel } from "../lib/format";
import { ApiError } from "../lib/api-client";

type HoldingTab = "all" | "spot" | "accounts" | "positions";

const tabs: Array<{ value: HoldingTab; label: string; kinds?: HoldingKind[] }> = [
  { value: "all", label: "全部" },
  { value: "spot", label: "Spot", kinds: ["spot"] },
  { value: "accounts", label: "合约余额", kinds: ["futures-balance", "contract-balance"] },
  { value: "positions", label: "仓位", kinds: ["futures-position", "contract-position"] },
];

function PnlValue({ value }: { value: string | number | undefined }) {
  const text = formatAmount(value);
  const negative = text.startsWith("-");
  return <span className={negative ? "number-negative" : text === "—" ? "muted-value" : "number-positive"}>{text}</span>;
}

export function HoldingsPage() {
  const { data, error, isError, isFetching, isLoading, refetch } = useSummary();
  const [tab, setTab] = useState<HoldingTab>("all");
  const [provider, setProvider] = useState("all");
  const [product, setProduct] = useState("all");
  const [search, setSearch] = useState("");

  const rows = useMemo(() => (data ? flattenHoldings(data) : []), [data]);
  const providerOptions = useMemo(
    () => Array.from(new Set(rows.map((row) => row.provider))).sort(),
    [rows],
  );
  const productOptions = useMemo(
    () => Array.from(new Set(rows.filter((row) => provider === "all" || row.provider === provider).map((row) => row.product))).sort(),
    [provider, rows],
  );
  const selectedTab = tabs.find((item) => item.value === tab) ?? tabs[0];
  const filteredRows = useMemo(() => {
    const query = search.trim().toLowerCase();
    return rows.filter((row) => {
      const matchesTab = !selectedTab.kinds || selectedTab.kinds.includes(row.kind);
      const matchesProvider = provider === "all" || row.provider === provider;
      const matchesProduct = product === "all" || row.product === product;
      const matchesSearch = !query || [row.provider, row.product, row.category, row.asset, row.symbol, row.side, row.marginType]
        .filter(Boolean)
        .join(" ")
        .toLowerCase()
        .includes(query);
      return matchesTab && matchesProvider && matchesProduct && matchesSearch;
    });
  }, [product, provider, rows, search, selectedTab.kinds]);

  const columns = useMemo<ColumnDef<HoldingRow, unknown>[]>(
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
      { accessorKey: "category", header: "类型" },
      {
        accessorKey: "symbol",
        header: "资产 / 交易对",
        cell: ({ row }) => (
          <div className="symbol-cell">
            <strong>{row.original.symbol || row.original.asset || "—"}</strong>
            {row.original.asset && row.original.asset !== row.original.symbol && <small>{row.original.asset}</small>}
          </div>
        ),
      },
      {
        id: "amount",
        header: "数量 / 余额",
        cell: ({ row }) => formatAmount(row.original.amount ?? row.original.total),
      },
      {
        id: "price",
        header: "开仓价",
        cell: ({ row }) => formatAmount(row.original.entryPrice),
      },
      {
        id: "mark",
        header: "标记价",
        cell: ({ row }) => formatAmount(row.original.markPrice),
      },
      {
        id: "pnl",
        header: "未实现盈亏",
        cell: ({ row }) => <PnlValue value={row.original.pnl} />,
      },
      {
        id: "position",
        header: "方向 / 杠杆",
        cell: ({ row }) => row.original.category === "仓位" ? (
          <div className="position-cell">
            <span>{row.original.side || "BOTH"}</span>
            {row.original.leverage && <small>{formatAmount(row.original.leverage, 2)}x</small>}
          </div>
        ) : <span className="muted-value">—</span>,
      },
    ],
    [],
  );

  const partialCount = data?.exchanges.filter((exchange) => exchange.status === "partial").length ?? 0;

  const selectChange = (setter: (value: string) => void) => (event: React.ChangeEvent<HTMLSelectElement>) => {
    setter(event.target.value);
    if (setter === setProvider) setProduct("all");
  };

  return (
    <>
      <PageHeader
        eyebrow="POSITIONS & BALANCES"
        title="持仓"
        description="查看各 Provider 的 Spot 余额、合约账户余额与当前仓位。"
        action={
          <Button disabled={isFetching} icon={<RefreshCw className={isFetching ? "animate-spin" : undefined} size={16} />} onClick={() => void refetch()}>
            刷新数据
          </Button>
        }
      />

      {isLoading && <DataState state="loading" />}
      {isError && <DataState message={error instanceof ApiError ? error.message : "持仓数据加载失败"} onRetry={() => void refetch()} state="error" />}
      {data && (
        <>
          {partialCount > 0 && (
            <div className="notice notice-warning" role="status">
              <SlidersHorizontal size={17} />
              <span>存在部分成功的 Provider。表格会保留已成功返回的条目，并标注对应来源状态。</span>
            </div>
          )}
          <section className="panel holdings-panel">
            <div className="holdings-tabs" role="tablist" aria-label="持仓类型">
              {tabs.map((item) => {
                const count = item.kinds ? rows.filter((row) => item.kinds?.includes(row.kind)).length : rows.length;
                return (
                  <button
                    aria-selected={tab === item.value}
                    className={tab === item.value ? "tab-button tab-button-active" : "tab-button"}
                    key={item.value}
                    onClick={() => setTab(item.value)}
                    role="tab"
                    type="button"
                  >
                    {item.label}<span>{count}</span>
                  </button>
                );
              })}
            </div>
            <div className="filter-toolbar">
              <div className="filter-search">
                <Search aria-hidden="true" size={16} />
                <input aria-label="搜索持仓" onChange={(event) => setSearch(event.target.value)} placeholder="搜索资产、交易对…" value={search} />
              </div>
              <label className="filter-select">
                <Filter size={14} />
                <span className="sr-only">Provider</span>
                <select aria-label="按 Provider 筛选" onChange={selectChange(setProvider)} value={provider}>
                  <option value="all">全部 Provider</option>
                  {providerOptions.map((item) => <option key={item} value={item}>{providerLabel(item)}</option>)}
                </select>
              </label>
              <label className="filter-select">
                <span className="sr-only">产品</span>
                <select aria-label="按产品筛选" onChange={selectChange(setProduct)} value={product}>
                  <option value="all">全部产品</option>
                  {productOptions.map((item) => <option key={item} value={item}>{productLabel(item)}</option>)}
                </select>
              </label>
            </div>
            <div className="table-toolbar-summary">
              <span><WalletCards size={15} />显示 {filteredRows.length} / {rows.length} 条</span>
              {(search || provider !== "all" || product !== "all" || tab !== "all") && <button className="clear-filter" onClick={() => { setSearch(""); setProvider("all"); setProduct("all"); setTab("all"); }} type="button">清除筛选</button>}
            </div>
            {!rows.length ? <DataState message="Provider 暂未返回持仓或余额数据" state="empty" /> : <DataTable caption="持仓与余额" columns={columns} data={filteredRows} emptyLabel="没有匹配当前筛选条件的条目" />}
          </section>
          <div className="legend-row">
            <StatusBadge compact status="ok" /> <span>数据来自 GET /api/v1/summary?include_zero=false</span>
          </div>
        </>
      )}
    </>
  );
}
