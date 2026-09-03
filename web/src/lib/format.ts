import type { Decimal, ExchangeData, ProductData, SummarySnapshot } from "../types/api";

export function textValue(value: Decimal | null | undefined): string {
  if (value === null || value === undefined) return "—";
  const text = String(value).trim();
  return text || "—";
}

export function formatAmount(value: Decimal | null | undefined, maximumFractionDigits = 8): string {
  if (value === null || value === undefined || String(value).trim() === "") return "—";
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) return String(value);
  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits,
    minimumFractionDigits: 0,
  }).format(parsed);
}

export function formatDate(value: string | number | undefined): string {
  if (!value) return "—";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return String(value);
  return new Intl.DateTimeFormat("zh-CN", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function statusLabel(status: string | undefined): string {
  switch (status) {
    case "ok":
      return "已连接";
    case "partial":
      return "部分成功";
    case "error":
      return "错误";
    default:
      return status || "未知";
  }
}

export function statusTone(status: string | undefined): "success" | "warning" | "danger" | "neutral" {
  switch (status) {
    case "ok":
      return "success";
    case "partial":
      return "warning";
    case "error":
      return "danger";
    default:
      return "neutral";
  }
}

export function providerLabel(provider: string): string {
  const labels: Record<string, string> = {
    binance: "Binance",
    bitget: "Bitget",
    weex: "Weex",
  };
  return labels[provider.toLowerCase()] ?? provider;
}

export function productLabel(product: string): string {
  const labels: Record<string, string> = {
    spot: "Spot",
    usdm: "USDⓈ-M Futures",
    usdcm: "USDC-M Futures",
    coinm: "COIN-M Futures",
    contract: "Contract",
  };
  return labels[product.toLowerCase()] ?? product.toUpperCase();
}

export function allProducts(snapshot: SummarySnapshot): Array<{ provider: string; product: ProductData }> {
  return snapshot.exchanges.flatMap((exchange) =>
    (exchange.products ?? []).map((product) => ({ provider: exchange.provider, product })),
  );
}

export function countPositions(exchange: ExchangeData): number {
  return (exchange.products ?? []).reduce(
    (total, product) => total + (product.futures_positions?.length ?? 0) + (product.contract_positions?.length ?? 0),
    0,
  );
}

export function countBalances(exchange: ExchangeData): number {
  return (
    (exchange.balances?.length ?? 0) +
    (exchange.products ?? []).reduce(
      (total, product) =>
        total +
        (product.futures_balances?.length ?? 0) +
        (product.contract_balances?.length ?? 0),
      0,
    )
  );
}
