/**
 * Types mirror the normalized summary response from internal/modules/summary.
 * Decimal values remain strings at the API boundary so the UI never rounds an
 * exchange value by accident.
 */
export type Decimal = string | number;

export interface Balance {
  symbol: string;
  free: Decimal;
  locked: Decimal;
  total: Decimal;
}

export interface FuturesAccountBalance {
  account_alias?: string;
  asset: string;
  balance: Decimal;
  withdraw_available?: Decimal;
  cross_wallet_balance?: Decimal;
  cross_unrealized_profit?: Decimal;
  available_balance?: Decimal;
  max_withdraw_amount?: Decimal;
  margin_available?: boolean;
  update_time?: number;
}

export interface FuturesPosition {
  symbol: string;
  pair?: string;
  position_side: string;
  position_amount: Decimal;
  entry_price: Decimal;
  break_even_price?: Decimal;
  mark_price: Decimal;
  unrealized_profit: Decimal;
  liquidation_price: Decimal;
  leverage: Decimal;
  margin_type?: string;
  isolated_margin?: Decimal;
  is_auto_add_margin?: boolean;
  notional?: Decimal;
  notional_value?: Decimal;
  margin_asset?: string;
  isolated_wallet?: Decimal;
  initial_margin?: Decimal;
  maintenance_margin?: Decimal;
  position_initial_margin?: Decimal;
  open_order_initial_margin?: Decimal;
  max_notional_value?: Decimal;
  max_quantity?: Decimal;
  bid_notional?: Decimal;
  ask_notional?: Decimal;
  adl?: number;
  adl_quantile?: number;
  update_time?: number;
}

export interface ContractPosition {
  id: number;
  asset: string;
  symbol: string;
  side: string;
  margin_type: string;
  separated_mode: string;
  separated_open_order_id: number;
  leverage: Decimal;
  size: Decimal;
  open_value: Decimal;
  open_fee: Decimal;
  funding_fee: Decimal;
  margin_size: Decimal;
  isolated_margin: Decimal;
  is_auto_append_isolated_margin: boolean;
  cum_open_size: Decimal;
  cum_open_value: Decimal;
  cum_open_fee: Decimal;
  cum_close_size: Decimal;
  cum_close_value: Decimal;
  cum_close_fee: Decimal;
  cum_funding_fee: Decimal;
  cum_liquidate_fee: Decimal;
  created_match_sequence_id: number;
  updated_match_sequence_id: number;
  created_time: number;
  updated_time: number;
  unrealize_pnl: Decimal;
  liquidate_price: Decimal;
}

export type CollectionStatus = "ok" | "partial" | "error" | string;

export interface ProductData {
  product: string;
  status: CollectionStatus;
  futures_balances?: FuturesAccountBalance[];
  futures_positions?: FuturesPosition[];
  contract_balances?: Balance[];
  contract_positions?: ContractPosition[];
  error?: string;
}

export interface ExchangeData {
  provider: string;
  status: CollectionStatus;
  balances?: Balance[];
  products?: ProductData[];
  error?: string;
}

export interface BankAccount {
  account_id: string;
  currency: string;
  balance: Decimal;
}

export interface BankData {
  provider: string;
  status: CollectionStatus;
  accounts?: BankAccount[];
  error?: string;
}

export interface SummarySnapshot {
  generated_at: string;
  exchanges: ExchangeData[];
  banks: BankData[];
}

export interface LoginResponse {
  x_token: string;
  expires_at: string;
}

export interface ErrorResponse {
  error?: {
    code?: string;
    message?: string;
    parameter?: string;
  };
  message?: string;
}
