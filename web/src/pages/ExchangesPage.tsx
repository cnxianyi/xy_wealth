import { useMemo } from "react";
import { ArrowUpRight, CircleAlert, Layers3, RefreshCw, Server, Wallet, Workflow } from "lucide-react";
import { PageHeader } from "../components/layout/PageHeader";
import { Button } from "../components/ui/Button";
import { DataState } from "../components/ui/DataState";
import { StatusBadge } from "../components/ui/StatusBadge";
import { useSummary } from "../features/summary/useSummary";
import { productRowCount } from "../features/summary/rows";
import { countBalances, countPositions } from "../lib/format";
import { productLabel, providerLabel } from "../lib/format";
import { ApiError } from "../lib/api-client";
import type { ExchangeData, ProductData } from "../types/api";

function ProductCard({ product }: { product: ProductData }) {
  const counts = productRowCount(product);
  return (
    <article className="product-card">
      <div className="product-card-heading">
        <div className="product-icon"><Workflow size={17} /></div>
        <div>
          <h3>{productLabel(product.product)}</h3>
          <p>{product.product === "contract" ? "Weex 合约账户" : "只读账户数据"}</p>
        </div>
        <StatusBadge compact status={product.status} />
      </div>
      <div className="product-metrics">
        <div><span>余额条目</span><strong>{counts.balances}</strong></div>
        <div><span>仓位</span><strong>{counts.positions}</strong></div>
      </div>
      {product.error && <p className="inline-error"><CircleAlert size={14} />{product.error}</p>}
    </article>
  );
}

function ExchangeCard({ exchange }: { exchange: ExchangeData }) {
  const products = exchange.products ?? [];
  const balances = countBalances(exchange);
  const positions = countPositions(exchange);
  const isBitget = exchange.provider.toLowerCase() === "bitget";
  return (
    <article className="exchange-card">
      <div className="exchange-card-heading">
        <div className="exchange-identity">
          <div className="provider-avatar provider-avatar-large">{providerLabel(exchange.provider).slice(0, 1)}</div>
          <div>
            <div className="exchange-name-row"><h2>{providerLabel(exchange.provider)}</h2><StatusBadge status={exchange.status} /></div>
            <p>只读连接 · {balances} 个余额条目 · {positions} 个仓位</p>
          </div>
        </div>
        <span className="exchange-arrow"><ArrowUpRight size={18} /></span>
      </div>
      {isBitget && (
        <div className="notice notice-info uta-note">
          <Layers3 size={17} />
          <span><strong>Bitget UTA 统一账户</strong>：合约余额来自统一资产账户视图，不代表 USDT、USDC、COIN 各有一份独立余额。</span>
        </div>
      )}
      {exchange.error && <div className="exchange-error"><CircleAlert size={16} /><span>{exchange.error}</span></div>}
      <div className="product-tree">
        <div className="product-tree-label"><span className="tree-line" /><Server size={15} />Spot</div>
        {products.map((product) => <ProductCard key={product.product} product={product} />)}
      </div>
    </article>
  );
}

export function ExchangesPage() {
  const { data, error, isError, isFetching, isLoading, refetch } = useSummary();
  const healthy = useMemo(() => data?.exchanges.filter((exchange) => exchange.status === "ok").length ?? 0, [data]);

  return (
    <>
      <PageHeader
        eyebrow="CONNECTED SOURCES"
        title="交易所"
        description="按 Provider 查看账户面与产品能力，保持上游错误透明可见。"
        action={<Button disabled={isFetching} icon={<RefreshCw className={isFetching ? "animate-spin" : undefined} size={16} />} onClick={() => void refetch()}>刷新数据</Button>}
      />
      {isLoading && <DataState state="loading" />}
      {isError && <DataState message={error instanceof ApiError ? error.message : "交易所数据加载失败"} onRetry={() => void refetch()} state="error" />}
      {data && (
        <>
          <div className="exchange-summary-strip">
            <div><span className="strip-icon"><Server size={16} /></span><span><strong>{data.exchanges.length}</strong> 个已配置 Provider</span></div>
            <div><span className="strip-icon strip-icon-green"><Wallet size={16} /></span><span><strong>{healthy}</strong> 个完全正常</span></div>
            <span className="strip-caption">数据源为实时请求，不在前端缓存密钥。</span>
          </div>
          {!data.exchanges.length ? <DataState message="当前没有已配置的交易所 Provider" state="empty" /> : (
            <div className="exchange-list">
              {data.exchanges.map((exchange) => <ExchangeCard exchange={exchange} key={exchange.provider} />)}
            </div>
          )}
          {data.exchanges.length > 0 && data.exchanges.every((exchange) => !exchange.balances?.length && !exchange.products?.length) && (
            <div className="panel exchange-empty-detail"><DataState message="Provider 已返回，但暂时没有余额或产品数据" state="empty" /></div>
          )}
          <p className="valuation-note"><Layers3 size={15} />各产品能力由后端 Provider 独立实现；未注册的能力不会在这里显示。</p>
        </>
      )}
    </>
  );
}
