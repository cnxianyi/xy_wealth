package exchange

import "context"

// USDSMFuturesProvider is the public, read-only USDⓈ-M Futures REST surface.
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

type USDSMFuturesExchangeInfo struct {
	Timezone        string                   `json:"timezone"`
	ServerTime      int64                    `json:"server_time,omitempty"`
	FuturesType     string                   `json:"futures_type,omitempty"`
	RateLimits      []RateLimit              `json:"rate_limits,omitempty"`
	ExchangeFilters []map[string]any         `json:"exchange_filters,omitempty"`
	Assets          []FuturesAsset           `json:"assets,omitempty"`
	Symbols         []USDSMFuturesSymbolInfo `json:"symbols"`
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
	WeightedAvgPrice   string `json:"weighted_avg_price"`
	LastPrice          string `json:"last_price"`
	LastQty            string `json:"last_qty"`
	OpenPrice          string `json:"open_price"`
	HighPrice          string `json:"high_price"`
	LowPrice           string `json:"low_price"`
	Volume             string `json:"volume"`
	BaseVolume         string `json:"base_volume,omitempty"`
	QuoteVolume        string `json:"quote_volume"`
	OpenTime           int64  `json:"open_time"`
	CloseTime          int64  `json:"close_time"`
	FirstID            int64  `json:"first_id"`
	LastID             int64  `json:"last_id"`
	Count              int64  `json:"count"`
}

type FuturesPremiumIndex struct {
	Symbol               string `json:"symbol"`
	MarkPrice            string `json:"mark_price"`
	IndexPrice           string `json:"index_price"`
	EstimatedSettlePrice string `json:"estimated_settle_price,omitempty"`
	LastFundingRate      string `json:"last_funding_rate,omitempty"`
	InterestRate         string `json:"interest_rate,omitempty"`
	NextFundingTime      int64  `json:"next_funding_time,omitempty"`
	Time                 int64  `json:"time"`
}
