import type { Balance, ContractPosition, ExchangeData, FuturesAccountBalance, FuturesPosition, ProductData, SummarySnapshot } from "../../types/api";

export type HoldingKind = "spot" | "futures-balance" | "futures-position" | "contract-balance" | "contract-position";

export interface HoldingRow {
  id: string;
  provider: string;
  product: string;
  kind: HoldingKind;
  category: "Spot" | "账户余额" | "仓位";
  asset: string;
  symbol: string;
  total?: string | number;
  available?: string | number;
  locked?: string | number;
  amount?: string | number;
  entryPrice?: string | number;
  markPrice?: string | number;
  pnl?: string | number;
  side?: string;
  leverage?: string | number;
  marginType?: string;
}

export interface OverviewBalanceRow {
  id: string;
  provider: string;
  product: string;
  symbol: string;
  total: string | number;
  available: string | number;
  locked?: string | number;
}

function spotRow(provider: string, product: string, balance: Balance, index: number): HoldingRow {
  return {
    id: `${provider}-${product}-spot-${balance.symbol}-${index}`,
    provider,
    product,
    kind: "spot",
    category: "Spot",
    asset: balance.symbol,
    symbol: balance.symbol,
    total: balance.total,
    available: balance.free,
    locked: balance.locked,
  };
}

function futuresBalanceRow(provider: string, product: string, balance: FuturesAccountBalance, index: number): HoldingRow {
  return {
    id: `${provider}-${product}-balance-${balance.asset}-${index}`,
    provider,
    product,
    kind: "futures-balance",
    category: "账户余额",
    asset: balance.asset,
    symbol: balance.asset,
    total: balance.balance,
    available: balance.available_balance ?? balance.withdraw_available,
    pnl: balance.cross_unrealized_profit,
  };
}

function contractBalanceRow(provider: string, product: string, balance: Balance, index: number): HoldingRow {
  return {
    id: `${provider}-${product}-contract-balance-${balance.symbol}-${index}`,
    provider,
    product,
    kind: "contract-balance",
    category: "账户余额",
    asset: balance.symbol,
    symbol: balance.symbol,
    total: balance.total,
    available: balance.free,
    locked: balance.locked,
  };
}

function futuresPositionRow(provider: string, product: string, position: FuturesPosition, index: number): HoldingRow {
  return {
    id: `${provider}-${product}-position-${position.symbol}-${index}`,
    provider,
    product,
    kind: "futures-position",
    category: "仓位",
    asset: position.margin_asset ?? "",
    symbol: position.symbol,
    amount: position.position_amount,
    entryPrice: position.entry_price,
    markPrice: position.mark_price,
    pnl: position.unrealized_profit,
    side: position.position_side,
    leverage: position.leverage,
    marginType: position.margin_type,
  };
}

function contractPositionRow(provider: string, product: string, position: ContractPosition, index: number): HoldingRow {
  return {
    id: `${provider}-${product}-contract-position-${position.id || position.symbol}-${index}`,
    provider,
    product,
    kind: "contract-position",
    category: "仓位",
    asset: position.asset,
    symbol: position.symbol,
    amount: position.size,
    entryPrice: position.open_value,
    markPrice: position.liquidate_price,
    pnl: position.unrealize_pnl,
    side: position.side,
    leverage: position.leverage,
    marginType: position.margin_type,
  };
}

export function flattenHoldings(snapshot: SummarySnapshot): HoldingRow[] {
  const rows: HoldingRow[] = [];
  snapshot.exchanges.forEach((exchange: ExchangeData) => {
    (exchange.balances ?? []).forEach((balance, index) => rows.push(spotRow(exchange.provider, "spot", balance, index)));
    (exchange.products ?? []).forEach((product: ProductData) => {
      (product.futures_balances ?? []).forEach((balance, index) => rows.push(futuresBalanceRow(exchange.provider, product.product, balance, index)));
      (product.futures_positions ?? []).forEach((position, index) => rows.push(futuresPositionRow(exchange.provider, product.product, position, index)));
      (product.contract_balances ?? []).forEach((balance, index) => rows.push(contractBalanceRow(exchange.provider, product.product, balance, index)));
      (product.contract_positions ?? []).forEach((position, index) => rows.push(contractPositionRow(exchange.provider, product.product, position, index)));
    });
  });
  return rows;
}

export function flattenOverviewBalances(snapshot: SummarySnapshot): OverviewBalanceRow[] {
  return flattenHoldings(snapshot)
    .filter((row) => row.kind === "spot" || row.kind === "futures-balance" || row.kind === "contract-balance")
    .map((row) => ({
      id: row.id,
      provider: row.provider,
      product: row.product,
      symbol: row.asset || row.symbol,
      total: row.total ?? "",
      available: row.available ?? "",
      locked: row.locked,
    }));
}

export function productRowCount(product: ProductData): { balances: number; positions: number } {
  return {
    balances: (product.futures_balances?.length ?? 0) + (product.contract_balances?.length ?? 0),
    positions: (product.futures_positions?.length ?? 0) + (product.contract_positions?.length ?? 0),
  };
}
