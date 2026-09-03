package exchange

import (
	"context"
	"fmt"
)

// InvalidParameterError indicates a parameter rejected before making an
// upstream request.
type InvalidParameterError struct {
	Parameter string
	Message   string
}

func (e InvalidParameterError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("invalid parameter %q", e.Parameter)
	}
	return fmt.Sprintf("invalid parameter %q: %s", e.Parameter, e.Message)
}

// SpotProvider is the read-only Spot REST surface shared by exchange
// integrations. Trading and account-write operations should be added as
// separate capabilities when their safety rules are defined.
type SpotProvider interface {
	Provider
	Ping(ctx context.Context) error
	ServerTime(ctx context.Context) (ServerTime, error)
	ExchangeInfo(ctx context.Context, symbol string) (ExchangeInfo, error)
	Depth(ctx context.Context, symbol string, limit int) (OrderBook, error)
	Klines(ctx context.Context, request KlinesRequest) ([]Kline, error)
	Ticker24hr(ctx context.Context, symbol string) (Ticker24hr, error)
	TickerPrice(ctx context.Context, symbol string) (PriceTicker, error)
	BookTicker(ctx context.Context, symbol string) (BookTicker, error)
}

type ServerTime struct {
	ServerTime int64 `json:"server_time"`
}

type ExchangeInfo struct {
	Timezone        string           `json:"timezone"`
	ServerTime      int64            `json:"server_time,omitempty"`
	RateLimits      []RateLimit      `json:"rate_limits,omitempty"`
	ExchangeFilters []map[string]any `json:"exchange_filters,omitempty"`
	Symbols         []SymbolInfo     `json:"symbols"`
}

type RateLimit struct {
	RateLimitType string `json:"rate_limit_type"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"interval_num"`
	Limit         int    `json:"limit"`
}

type SymbolInfo struct {
	Symbol                     string           `json:"symbol"`
	Status                     string           `json:"status"`
	BaseAsset                  string           `json:"base_asset"`
	BaseAssetPrecision         int              `json:"base_asset_precision"`
	QuoteAsset                 string           `json:"quote_asset"`
	QuotePrecision             int              `json:"quote_precision"`
	QuoteAssetPrecision        int              `json:"quote_asset_precision"`
	TickSize                   string           `json:"tick_size,omitempty"`
	StepSize                   string           `json:"step_size,omitempty"`
	MinTradeAmount             string           `json:"min_trade_amount,omitempty"`
	MaxTradeAmount             string           `json:"max_trade_amount,omitempty"`
	TakerFeeRate               string           `json:"taker_fee_rate,omitempty"`
	MakerFeeRate               string           `json:"maker_fee_rate,omitempty"`
	BuyLimitPriceRatio         string           `json:"buy_limit_price_ratio,omitempty"`
	SellLimitPriceRatio        string           `json:"sell_limit_price_ratio,omitempty"`
	MarketBuyLimitSize         string           `json:"market_buy_limit_size,omitempty"`
	MarketSellLimitSize        string           `json:"market_sell_limit_size,omitempty"`
	MarketFallbackPriceRatio   string           `json:"market_fallback_price_ratio,omitempty"`
	EnableTrade                *bool            `json:"enable_trade,omitempty"`
	EnableDisplay              *bool            `json:"enable_display,omitempty"`
	DisplayDigitMerge          string           `json:"display_digit_merge,omitempty"`
	DisplayNew                 *bool            `json:"display_new,omitempty"`
	DisplayHot                 *bool            `json:"display_hot,omitempty"`
	BaseCommissionPrecision    int              `json:"base_commission_precision,omitempty"`
	QuoteCommissionPrecision   int              `json:"quote_commission_precision,omitempty"`
	OrderTypes                 []string         `json:"order_types,omitempty"`
	IcebergAllowed             bool             `json:"iceberg_allowed,omitempty"`
	OCOAllowed                 bool             `json:"oco_allowed,omitempty"`
	QuoteOrderQtyMarketAllowed bool             `json:"quote_order_qty_market_allowed,omitempty"`
	IsSpotTradingAllowed       bool             `json:"is_spot_trading_allowed,omitempty"`
	IsMarginTradingAllowed     bool             `json:"is_margin_trading_allowed,omitempty"`
	Filters                    []map[string]any `json:"filters,omitempty"`
	Permissions                []string         `json:"permissions,omitempty"`
	PermissionSets             [][]string       `json:"permission_sets,omitempty"`
}

type OrderBook struct {
	LastUpdateID int64      `json:"last_update_id"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type KlinesRequest struct {
	Symbol    string
	Interval  string
	StartTime *int64
	EndTime   *int64
	TimeZone  string
	Limit     int
}

type Kline struct {
	OpenTime                 int64  `json:"open_time"`
	Open                     string `json:"open"`
	High                     string `json:"high"`
	Low                      string `json:"low"`
	Close                    string `json:"close"`
	Volume                   string `json:"volume"`
	CloseTime                int64  `json:"close_time"`
	QuoteAssetVolume         string `json:"quote_asset_volume"`
	NumberOfTrades           int64  `json:"number_of_trades"`
	TakerBuyBaseAssetVolume  string `json:"taker_buy_base_asset_volume"`
	TakerBuyQuoteAssetVolume string `json:"taker_buy_quote_asset_volume"`
	Ignore                   string `json:"ignore"`
}

type Ticker24hr struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"price_change"`
	PriceChangePercent string `json:"price_change_percent"`
	WeightedAvgPrice   string `json:"weighted_avg_price,omitempty"`
	PrevClosePrice     string `json:"prev_close_price,omitempty"`
	LastPrice          string `json:"last_price"`
	LastQty            string `json:"last_qty,omitempty"`
	BidPrice           string `json:"bid_price"`
	BidQty             string `json:"bid_qty"`
	AskPrice           string `json:"ask_price"`
	AskQty             string `json:"ask_qty"`
	OpenPrice          string `json:"open_price"`
	HighPrice          string `json:"high_price"`
	LowPrice           string `json:"low_price"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quote_volume"`
	OpenTime           int64  `json:"open_time"`
	CloseTime          int64  `json:"close_time"`
	FirstID            int64  `json:"first_id,omitempty"`
	LastID             int64  `json:"last_id,omitempty"`
	Count              int64  `json:"count"`
}

type PriceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type BookTicker struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bid_price"`
	BidQty   string `json:"bid_qty"`
	AskPrice string `json:"ask_price"`
	AskQty   string `json:"ask_qty"`
}
