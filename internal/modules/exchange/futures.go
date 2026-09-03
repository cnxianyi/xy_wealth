package exchange

import (
	"context"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
)

// USDSMFuturesProvider is the public, read-only USDⓈ-M Futures / compatible
// Contract REST surface.
// Trading, account and listen-key operations are separate capabilities and
// are intentionally not part of this initial integration.
type USDSMFuturesProvider interface {
	Provider
	FuturesPing(ctx context.Context) error
	FuturesServerTime(ctx context.Context) (ServerTime, error)
	FuturesExchangeInfo(ctx context.Context) (USDSMFuturesExchangeInfo, error)
	FuturesDepth(ctx context.Context, symbol string, limit int) (FuturesOrderBook, error)
	FuturesKlines(ctx context.Context, request KlinesRequest) ([]Kline, error)
	FuturesTicker24hr(ctx context.Context, symbol string) (FuturesTicker24hr, error)
	FuturesTickerPrice(ctx context.Context, symbol string) (PriceTicker, error)
	FuturesBookTicker(ctx context.Context, symbol string) (BookTicker, error)
	FuturesPremiumIndex(ctx context.Context, symbol string) (FuturesPremiumIndex, error)
}

// ContractPositionProvider exposes read-only Contract position data for
// providers that implement a signed position endpoint. It is separate from
// USDSMFuturesProvider so providers without a position API are not registered
// as if that capability existed.
type ContractPositionProvider interface {
	Provider
	ContractPositions(ctx context.Context, symbol string) ([]ContractPosition, error)
}

// ContractBalanceProvider exposes read-only Contract account balances for
// providers that implement a signed balance endpoint. It is separate from
// Provider.Balances, which represents the provider's default (Spot) account.
type ContractBalanceProvider interface {
	Provider
	ContractBalances(ctx context.Context) ([]asset.Balance, error)
}

// USDSMFuturesAccountProvider exposes signed USDⓈ-M/USDT-FUTURES account data
// for providers that implement this capability.
// It is separate from the public market provider so account credentials and
// signed endpoints are not implied by a market-data implementation.
type USDSMFuturesAccountProvider interface {
	Provider
	USDSMFuturesAccountBalances(ctx context.Context) ([]FuturesAccountBalance, error)
	USDSMFuturesPositions(ctx context.Context, symbol string) ([]FuturesPosition, error)
}

// COINMFuturesAccountProvider exposes Binance COIN-M Futures account data.
type COINMFuturesAccountProvider interface {
	Provider
	COINMFuturesAccountBalances(ctx context.Context) ([]FuturesAccountBalance, error)
	COINMFuturesPositions(ctx context.Context, symbol string) ([]FuturesPosition, error)
}

// COINMFuturesProvider is the public, read-only COIN-M Futures REST surface.
// Trading, account and listen-key operations are separate capabilities and
// are intentionally not part of this initial integration.
type COINMFuturesProvider interface {
	Provider
	CoinMFuturesPing(ctx context.Context) error
	CoinMFuturesServerTime(ctx context.Context) (ServerTime, error)
	CoinMFuturesExchangeInfo(ctx context.Context) (COINMFuturesExchangeInfo, error)
	CoinMFuturesDepth(ctx context.Context, symbol string, limit int) (FuturesOrderBook, error)
	CoinMFuturesKlines(ctx context.Context, request KlinesRequest) ([]Kline, error)
	CoinMFuturesTicker24hr(ctx context.Context, symbol string) (COINMFuturesTicker24hr, error)
	CoinMFuturesTickerPrice(ctx context.Context, symbol string) (COINMFuturesPriceTicker, error)
	CoinMFuturesBookTicker(ctx context.Context, symbol string) (COINMFuturesBookTicker, error)
	CoinMFuturesPremiumIndex(ctx context.Context, symbol string) (COINMFuturesPremiumIndex, error)
}

type USDSMFuturesExchangeInfo struct {
	Timezone        string                   `json:"timezone"`
	ServerTime      int64                    `json:"server_time,omitempty"`
	FuturesType     string                   `json:"futures_type,omitempty"`
	RateLimits      []RateLimit              `json:"rate_limits,omitempty"`
	ExchangeFilters []map[string]any         `json:"exchange_filters,omitempty"`
	Assets          []FuturesAsset           `json:"assets,omitempty"`
	Symbols         []USDSMFuturesSymbolInfo `json:"symbols"`
}

type COINMFuturesExchangeInfo struct {
	Timezone        string                   `json:"timezone"`
	ServerTime      int64                    `json:"server_time,omitempty"`
	RateLimits      []RateLimit              `json:"rate_limits,omitempty"`
	ExchangeFilters []map[string]any         `json:"exchange_filters,omitempty"`
	Symbols         []COINMFuturesSymbolInfo `json:"symbols"`
}

type COINMFuturesSymbolInfo struct {
	Symbol                string           `json:"symbol"`
	Pair                  string           `json:"pair"`
	ContractType          string           `json:"contract_type"`
	DeliveryDate          int64            `json:"delivery_date"`
	OnboardDate           int64            `json:"onboard_date"`
	ContractStatus        string           `json:"contract_status"`
	MaintMarginPercent    string           `json:"maint_margin_percent,omitempty"`
	RequiredMarginPercent string           `json:"required_margin_percent,omitempty"`
	BaseAsset             string           `json:"base_asset"`
	QuoteAsset            string           `json:"quote_asset"`
	MarginAsset           string           `json:"margin_asset"`
	PricePrecision        int              `json:"price_precision"`
	QuantityPrecision     int              `json:"quantity_precision"`
	BaseAssetPrecision    int              `json:"base_asset_precision"`
	QuotePrecision        int              `json:"quote_precision"`
	UnderlyingType        string           `json:"underlying_type,omitempty"`
	UnderlyingSubType     []string         `json:"underlying_sub_type,omitempty"`
	EqualQtyPrecision     int              `json:"equal_qty_precision,omitempty"`
	TriggerProtect        string           `json:"trigger_protect,omitempty"`
	LiquidationFee        string           `json:"liquidation_fee,omitempty"`
	MarketTakeBound       string           `json:"market_take_bound,omitempty"`
	MaxMoveOrderLimit     int              `json:"max_move_order_limit,omitempty"`
	ContractSize          int64            `json:"contract_size,omitempty"`
	OrderTypes            []string         `json:"order_types,omitempty"`
	TimeInForce           []string         `json:"time_in_force,omitempty"`
	PermissionSets        []string         `json:"permission_sets,omitempty"`
	Filters               []map[string]any `json:"filters,omitempty"`
}

type FuturesAsset struct {
	Asset             string `json:"asset"`
	MarginAvailable   bool   `json:"margin_available"`
	AutoAssetExchange string `json:"auto_asset_exchange,omitempty"`
}

type USDSMFuturesSymbolInfo struct {
	Symbol                string           `json:"symbol"`
	Pair                  string           `json:"pair"`
	ContractType          string           `json:"contract_type"`
	DisplaySymbol         string           `json:"display_symbol,omitempty"`
	DeliveryDate          int64            `json:"delivery_date"`
	OnboardDate           int64            `json:"onboard_date"`
	Status                string           `json:"status"`
	MaintMarginPercent    string           `json:"maint_margin_percent,omitempty"`
	RequiredMarginPercent string           `json:"required_margin_percent,omitempty"`
	BaseAsset             string           `json:"base_asset"`
	QuoteAsset            string           `json:"quote_asset"`
	MarginAsset           string           `json:"margin_asset"`
	PricePrecision        int              `json:"price_precision"`
	QuantityPrecision     int              `json:"quantity_precision"`
	BaseAssetPrecision    int              `json:"base_asset_precision"`
	QuotePrecision        int              `json:"quote_precision"`
	UnderlyingType        string           `json:"underlying_type,omitempty"`
	UnderlyingSubType     []string         `json:"underlying_sub_type,omitempty"`
	ContractVal           string           `json:"contract_val,omitempty"`
	Delivery              []string         `json:"delivery,omitempty"`
	ForwardContractFlag   bool             `json:"forward_contract_flag,omitempty"`
	MinLeverage           int              `json:"min_leverage,omitempty"`
	MaxLeverage           int              `json:"max_leverage,omitempty"`
	BuyLimitPriceRatio    string           `json:"buy_limit_price_ratio,omitempty"`
	SellLimitPriceRatio   string           `json:"sell_limit_price_ratio,omitempty"`
	MinTradeUSDT          string           `json:"min_trade_usdt,omitempty"`
	MakerFeeRate          string           `json:"maker_fee_rate,omitempty"`
	TakerFeeRate          string           `json:"taker_fee_rate,omitempty"`
	APIMakerFeeRate       string           `json:"api_maker_fee_rate,omitempty"`
	APITakerFeeRate       string           `json:"api_taker_fee_rate,omitempty"`
	MinOrderSize          string           `json:"min_order_size,omitempty"`
	MaxOrderSize          string           `json:"max_order_size,omitempty"`
	MaxPositionSize       string           `json:"max_position_size,omitempty"`
	MarketOpenLimitSize   string           `json:"market_open_limit_size,omitempty"`
	SettlePlan            int              `json:"settle_plan,omitempty"`
	TriggerProtect        string           `json:"trigger_protect,omitempty"`
	LiquidationFee        string           `json:"liquidation_fee,omitempty"`
	MarketTakeBound       string           `json:"market_take_bound,omitempty"`
	MaxMoveOrderLimit     int              `json:"max_move_order_limit,omitempty"`
	OrderTypes            []string         `json:"order_types,omitempty"`
	TimeInForce           []string         `json:"time_in_force,omitempty"`
	PermissionSets        []string         `json:"permission_sets,omitempty"`
	Filters               []map[string]any `json:"filters,omitempty"`
}

type FuturesOrderBook struct {
	LastUpdateID    int64      `json:"last_update_id"`
	Symbol          string     `json:"symbol,omitempty"`
	Pair            string     `json:"pair,omitempty"`
	EventTime       int64      `json:"event_time,omitempty"`
	TransactionTime int64      `json:"transaction_time,omitempty"`
	Bids            [][]string `json:"bids"`
	Asks            [][]string `json:"asks"`
}

type FuturesTicker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"price_change"`
	PriceChangePercent string `json:"price_change_percent"`
	WeightedAvgPrice   string `json:"weighted_avg_price,omitempty"`
	LastPrice          string `json:"last_price"`
	LastQty            string `json:"last_qty,omitempty"`
	OpenPrice          string `json:"open_price"`
	HighPrice          string `json:"high_price"`
	LowPrice           string `json:"low_price"`
	Volume             string `json:"volume"`
	BaseVolume         string `json:"base_volume,omitempty"`
	QuoteVolume        string `json:"quote_volume"`
	MarkPrice          string `json:"mark_price,omitempty"`
	IndexPrice         string `json:"index_price,omitempty"`
	OpenTime           int64  `json:"open_time"`
	CloseTime          int64  `json:"close_time"`
	FirstID            int64  `json:"first_id,omitempty"`
	LastID             int64  `json:"last_id,omitempty"`
	Count              int64  `json:"count,omitempty"`
}

type FuturesPremiumIndex struct {
	Symbol               string `json:"symbol"`
	MarkPrice            string `json:"mark_price"`
	IndexPrice           string `json:"index_price"`
	EstimatedSettlePrice string `json:"estimated_settle_price,omitempty"`
	LastFundingRate      string `json:"last_funding_rate,omitempty"`
	ForecastFundingRate  string `json:"forecast_funding_rate,omitempty"`
	InterestRate         string `json:"interest_rate,omitempty"`
	NextFundingTime      int64  `json:"next_funding_time,omitempty"`
	Time                 int64  `json:"time"`
	CollectCycle         int64  `json:"collect_cycle,omitempty"`
}

// ContractPosition describes one Weex Contract position, including its
// direction, margin mode, size, fees, and unrealized PnL.
type ContractPosition struct {
	ID                         int64  `json:"id"`
	Asset                      string `json:"asset"`
	Symbol                     string `json:"symbol"`
	Side                       string `json:"side"`
	MarginType                 string `json:"margin_type"`
	SeparatedMode              string `json:"separated_mode"`
	SeparatedOpenOrderID       int64  `json:"separated_open_order_id"`
	Leverage                   string `json:"leverage"`
	Size                       string `json:"size"`
	OpenValue                  string `json:"open_value"`
	OpenFee                    string `json:"open_fee"`
	FundingFee                 string `json:"funding_fee"`
	MarginSize                 string `json:"margin_size"`
	IsolatedMargin             string `json:"isolated_margin"`
	IsAutoAppendIsolatedMargin bool   `json:"is_auto_append_isolated_margin"`
	CumOpenSize                string `json:"cum_open_size"`
	CumOpenValue               string `json:"cum_open_value"`
	CumOpenFee                 string `json:"cum_open_fee"`
	CumCloseSize               string `json:"cum_close_size"`
	CumCloseValue              string `json:"cum_close_value"`
	CumCloseFee                string `json:"cum_close_fee"`
	CumFundingFee              string `json:"cum_funding_fee"`
	CumLiquidateFee            string `json:"cum_liquidate_fee"`
	CreatedMatchSequenceID     int64  `json:"created_match_sequence_id"`
	UpdatedMatchSequenceID     int64  `json:"updated_match_sequence_id"`
	CreatedTime                int64  `json:"created_time"`
	UpdatedTime                int64  `json:"updated_time"`
	UnrealizePnl               string `json:"unrealize_pnl"`
	LiquidatePrice             string `json:"liquidate_price"`
}

// FuturesAccountBalance is the normalized account asset returned by futures
// balance endpoints. Decimal values remain strings to preserve exact exchange
// precision.
type FuturesAccountBalance struct {
	AccountAlias          string `json:"account_alias,omitempty"`
	Asset                 string `json:"asset"`
	Balance               string `json:"balance"`
	WithdrawAvailable     string `json:"withdraw_available,omitempty"`
	CrossWalletBalance    string `json:"cross_wallet_balance,omitempty"`
	CrossUnrealizedProfit string `json:"cross_unrealized_profit,omitempty"`
	AvailableBalance      string `json:"available_balance,omitempty"`
	MaxWithdrawAmount     string `json:"max_withdraw_amount,omitempty"`
	MarginAvailable       *bool  `json:"margin_available,omitempty"`
	UpdateTime            int64  `json:"update_time"`
}

// FuturesPosition is the normalized read-only position view shared by
// Binance USDⓈ-M and COIN-M Futures. Fields not supplied by a product are
// omitted from the JSON response.
type FuturesPosition struct {
	Symbol                 string `json:"symbol"`
	Pair                   string `json:"pair,omitempty"`
	PositionSide           string `json:"position_side"`
	PositionAmount         string `json:"position_amount"`
	EntryPrice             string `json:"entry_price"`
	BreakEvenPrice         string `json:"break_even_price,omitempty"`
	MarkPrice              string `json:"mark_price"`
	UnrealizedProfit       string `json:"unrealized_profit"`
	LiquidationPrice       string `json:"liquidation_price"`
	Leverage               string `json:"leverage"`
	MarginType             string `json:"margin_type,omitempty"`
	IsolatedMargin         string `json:"isolated_margin,omitempty"`
	IsAutoAddMargin        *bool  `json:"is_auto_add_margin,omitempty"`
	Notional               string `json:"notional,omitempty"`
	NotionalValue          string `json:"notional_value,omitempty"`
	MarginAsset            string `json:"margin_asset,omitempty"`
	IsolatedWallet         string `json:"isolated_wallet,omitempty"`
	InitialMargin          string `json:"initial_margin,omitempty"`
	MaintenanceMargin      string `json:"maintenance_margin,omitempty"`
	PositionInitialMargin  string `json:"position_initial_margin,omitempty"`
	OpenOrderInitialMargin string `json:"open_order_initial_margin,omitempty"`
	MaxNotionalValue       string `json:"max_notional_value,omitempty"`
	MaxQuantity            string `json:"max_quantity,omitempty"`
	BidNotional            string `json:"bid_notional,omitempty"`
	AskNotional            string `json:"ask_notional,omitempty"`
	ADL                    int64  `json:"adl,omitempty"`
	ADLQuantile            int64  `json:"adl_quantile,omitempty"`
	UpdateTime             int64  `json:"update_time"`
}

type COINMFuturesTicker24hr struct {
	Symbol             string `json:"symbol"`
	Pair               string `json:"pair"`
	PriceChange        string `json:"price_change"`
	PriceChangePercent string `json:"price_change_percent"`
	WeightedAvgPrice   string `json:"weighted_avg_price"`
	LastPrice          string `json:"last_price"`
	LastQty            string `json:"last_qty"`
	OpenPrice          string `json:"open_price"`
	HighPrice          string `json:"high_price"`
	LowPrice           string `json:"low_price"`
	Volume             string `json:"volume"`
	BaseVolume         string `json:"base_volume,omitempty"`
	OpenTime           int64  `json:"open_time"`
	CloseTime          int64  `json:"close_time"`
	FirstID            int64  `json:"first_id"`
	LastID             int64  `json:"last_id"`
	Count              int64  `json:"count"`
}

type COINMFuturesPriceTicker struct {
	Symbol string `json:"symbol"`
	Pair   string `json:"pair"`
	Price  string `json:"price"`
	Time   int64  `json:"time,omitempty"`
}

type COINMFuturesBookTicker struct {
	LastUpdateID int64  `json:"last_update_id"`
	Symbol       string `json:"symbol"`
	Pair         string `json:"pair"`
	BidPrice     string `json:"bid_price"`
	BidQty       string `json:"bid_qty"`
	AskPrice     string `json:"ask_price"`
	AskQty       string `json:"ask_qty"`
	Time         int64  `json:"time,omitempty"`
}

type COINMFuturesPremiumIndex struct {
	Symbol               string `json:"symbol"`
	Pair                 string `json:"pair"`
	MarkPrice            string `json:"mark_price"`
	IndexPrice           string `json:"index_price"`
	EstimatedSettlePrice string `json:"estimated_settle_price,omitempty"`
	LastFundingRate      string `json:"last_funding_rate,omitempty"`
	InterestRate         string `json:"interest_rate,omitempty"`
	NextFundingTime      int64  `json:"next_funding_time,omitempty"`
	Time                 int64  `json:"time"`
}
