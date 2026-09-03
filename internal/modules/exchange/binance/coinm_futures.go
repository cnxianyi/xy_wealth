package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

const (
	defaultCoinMFuturesDepthLimit  = 500
	defaultCoinMFuturesKlinesLimit = 500
	maxCoinMFuturesKlinesLimit     = 1500
)

var validCoinMFuturesDepthLimits = map[int]struct{}{
	5: {}, 10: {}, 20: {}, 50: {}, 100: {}, 500: {}, 1000: {},
}

type coinMFuturesExchangeInfoResponse struct {
	Timezone        string                       `json:"timezone"`
	ServerTime      int64                        `json:"serverTime"`
	RateLimits      []rateLimitResponse          `json:"rateLimits"`
	ExchangeFilters []map[string]any             `json:"exchangeFilters"`
	Symbols         []coinMFuturesSymbolResponse `json:"symbols"`
}

type coinMFuturesSymbolResponse struct {
	Symbol                string           `json:"symbol"`
	Pair                  string           `json:"pair"`
	ContractType          string           `json:"contractType"`
	DeliveryDate          int64            `json:"deliveryDate"`
	OnboardDate           int64            `json:"onboardDate"`
	ContractStatus        string           `json:"contractStatus"`
	MaintMarginPercent    string           `json:"maintMarginPercent"`
	RequiredMarginPercent string           `json:"requiredMarginPercent"`
	BaseAsset             string           `json:"baseAsset"`
	QuoteAsset            string           `json:"quoteAsset"`
	MarginAsset           string           `json:"marginAsset"`
	PricePrecision        int              `json:"pricePrecision"`
	QuantityPrecision     int              `json:"quantityPrecision"`
	BaseAssetPrecision    int              `json:"baseAssetPrecision"`
	QuotePrecision        int              `json:"quotePrecision"`
	UnderlyingType        string           `json:"underlyingType"`
	UnderlyingSubType     []string         `json:"underlyingSubType"`
	EqualQtyPrecision     int              `json:"equalQtyPrecision"`
	TriggerProtect        string           `json:"triggerProtect"`
	LiquidationFee        string           `json:"liquidationFee"`
	MarketTakeBound       string           `json:"marketTakeBound"`
	MaxMoveOrderLimit     int              `json:"maxMoveOrderLimit"`
	ContractSize          int64            `json:"contractSize"`
	OrderTypes            []string         `json:"orderTypes"`
	TimeInForce           []string         `json:"timeInForce"`
	PermissionSets        []string         `json:"permissionSets"`
	Filters               []map[string]any `json:"filters"`
}

type coinMFuturesTicker24hrResponse struct {
	Symbol             string `json:"symbol"`
	Pair               string `json:"pair"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	BaseVolume         string `json:"baseVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstID            int64  `json:"firstId"`
	LastID             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

type coinMFuturesPriceTickerResponse struct {
	Symbol string `json:"symbol"`
	Pair   string `json:"ps"`
	Price  string `json:"price"`
	Time   int64  `json:"time"`
}

type coinMFuturesBookTickerResponse struct {
	LastUpdateID int64  `json:"lastUpdateId"`
	Symbol       string `json:"symbol"`
	Pair         string `json:"pair"`
	BidPrice     string `json:"bidPrice"`
	BidQty       string `json:"bidQty"`
	AskPrice     string `json:"askPrice"`
	AskQty       string `json:"askQty"`
	Time         int64  `json:"time"`
}

type coinMFuturesPremiumIndexResponse struct {
	Symbol               string `json:"symbol"`
	Pair                 string `json:"pair"`
	MarkPrice            string `json:"markPrice"`
	IndexPrice           string `json:"indexPrice"`
	EstimatedSettlePrice string `json:"estimatedSettlePrice"`
	LastFundingRate      string `json:"lastFundingRate"`
	InterestRate         string `json:"interestRate"`
	NextFundingTime      int64  `json:"nextFundingTime"`
	Time                 int64  `json:"time"`
}

// CoinMFuturesPing tests connectivity to Binance's COIN-M Futures REST API.
func (c *Client) CoinMFuturesPing(ctx context.Context) error {
	return c.getCoinMFuturesJSON(ctx, "/dapi/v1/ping", nil, nil)
}

// CoinMFuturesServerTime returns Binance COIN-M Futures server time in milliseconds.
func (c *Client) CoinMFuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	return exchange.ServerTime{ServerTime: response.ServerTime}, nil
}

// CoinMFuturesExchangeInfo returns COIN-M Futures trading rules and contracts.
func (c *Client) CoinMFuturesExchangeInfo(ctx context.Context) (exchange.COINMFuturesExchangeInfo, error) {
	var response coinMFuturesExchangeInfoResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/exchangeInfo", nil, &response); err != nil {
		return exchange.COINMFuturesExchangeInfo{}, err
	}

	info := exchange.COINMFuturesExchangeInfo{
		Timezone:        response.Timezone,
		ServerTime:      response.ServerTime,
		ExchangeFilters: response.ExchangeFilters,
		RateLimits:      make([]exchange.RateLimit, 0, len(response.RateLimits)),
		Symbols:         make([]exchange.COINMFuturesSymbolInfo, 0, len(response.Symbols)),
	}
	for _, limit := range response.RateLimits {
		info.RateLimits = append(info.RateLimits, exchange.RateLimit{
			RateLimitType: limit.RateLimitType,
			Interval:      limit.Interval,
			IntervalNum:   limit.IntervalNum,
			Limit:         limit.Limit,
		})
	}
	for _, symbol := range response.Symbols {
		info.Symbols = append(info.Symbols, exchange.COINMFuturesSymbolInfo{
			Symbol:                symbol.Symbol,
			Pair:                  symbol.Pair,
			ContractType:          symbol.ContractType,
			DeliveryDate:          symbol.DeliveryDate,
			OnboardDate:           symbol.OnboardDate,
			ContractStatus:        symbol.ContractStatus,
			MaintMarginPercent:    symbol.MaintMarginPercent,
			RequiredMarginPercent: symbol.RequiredMarginPercent,
			BaseAsset:             symbol.BaseAsset,
			QuoteAsset:            symbol.QuoteAsset,
			MarginAsset:           symbol.MarginAsset,
			PricePrecision:        symbol.PricePrecision,
			QuantityPrecision:     symbol.QuantityPrecision,
			BaseAssetPrecision:    symbol.BaseAssetPrecision,
			QuotePrecision:        symbol.QuotePrecision,
			UnderlyingType:        symbol.UnderlyingType,
			UnderlyingSubType:     symbol.UnderlyingSubType,
			EqualQtyPrecision:     symbol.EqualQtyPrecision,
			TriggerProtect:        symbol.TriggerProtect,
			LiquidationFee:        symbol.LiquidationFee,
			MarketTakeBound:       symbol.MarketTakeBound,
			MaxMoveOrderLimit:     symbol.MaxMoveOrderLimit,
			ContractSize:          symbol.ContractSize,
			OrderTypes:            symbol.OrderTypes,
			TimeInForce:           symbol.TimeInForce,
			PermissionSets:        symbol.PermissionSets,
			Filters:               symbol.Filters,
		})
	}
	return info, nil
}

func (c *Client) CoinMFuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	if limit == 0 {
		limit = defaultCoinMFuturesDepthLimit
	}
	if _, ok := validCoinMFuturesDepthLimits[limit]; !ok {
		return exchange.FuturesOrderBook{}, exchange.InvalidParameterError{
			Parameter: "limit",
			Message:   "must be one of 5, 10, 20, 50, 100, 500, or 1000",
		}
	}

	query := url.Values{"symbol": []string{normalized}, "limit": []string{strconv.Itoa(limit)}}
	var response futuresOrderBookResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/depth", query, &response); err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	return exchange.FuturesOrderBook{
		LastUpdateID:    response.LastUpdateID,
		Symbol:          response.Symbol,
		Pair:            response.Pair,
		EventTime:       response.EventTime,
		TransactionTime: response.TransactionTime,
		Bids:            response.Bids,
		Asks:            response.Asks,
	}, nil
}

func (c *Client) CoinMFuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(request.Interval)
	if _, ok := validFuturesIntervals[interval]; !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported COIN-M Futures interval"}
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultCoinMFuturesKlinesLimit
	}
	if err := validateLimit("limit", limit, maxCoinMFuturesKlinesLimit); err != nil {
		return nil, err
	}
	if request.StartTime != nil && *request.StartTime < 0 {
		return nil, exchange.InvalidParameterError{Parameter: "startTime", Message: "must be non-negative"}
	}
	if request.EndTime != nil && *request.EndTime < 0 {
		return nil, exchange.InvalidParameterError{Parameter: "endTime", Message: "must be non-negative"}
	}
	if request.StartTime != nil && request.EndTime != nil && *request.StartTime > *request.EndTime {
		return nil, exchange.InvalidParameterError{Parameter: "startTime", Message: "must not be after endTime"}
	}

	query := url.Values{
		"symbol":   []string{normalized},
		"interval": []string{interval},
		"limit":    []string{strconv.Itoa(limit)},
	}
	if request.StartTime != nil {
		query.Set("startTime", strconv.FormatInt(*request.StartTime, 10))
	}
	if request.EndTime != nil {
		query.Set("endTime", strconv.FormatInt(*request.EndTime, 10))
	}

	var raw []json.RawMessage
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/klines", query, &raw); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(raw))
	for index, item := range raw {
		kline, err := decodeKline(item)
		if err != nil {
			return nil, fmt.Errorf("decode COIN-M Futures kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	return klines, nil
}

func (c *Client) CoinMFuturesTicker24hr(ctx context.Context, symbol string) (exchange.COINMFuturesTicker24hr, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.COINMFuturesTicker24hr{}, err
	}
	var responses []coinMFuturesTicker24hrResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/ticker/24hr", url.Values{"symbol": []string{normalized}}, &responses); err != nil {
		return exchange.COINMFuturesTicker24hr{}, err
	}
	if len(responses) == 0 {
		return exchange.COINMFuturesTicker24hr{}, fmt.Errorf("decode COIN-M Futures 24-hour ticker response: empty response")
	}
	response := responses[0]
	return exchange.COINMFuturesTicker24hr{
		Symbol:             response.Symbol,
		Pair:               response.Pair,
		PriceChange:        response.PriceChange,
		PriceChangePercent: response.PriceChangePercent,
		WeightedAvgPrice:   response.WeightedAvgPrice,
		LastPrice:          response.LastPrice,
		LastQty:            response.LastQty,
		OpenPrice:          response.OpenPrice,
		HighPrice:          response.HighPrice,
		LowPrice:           response.LowPrice,
		Volume:             response.Volume,
		BaseVolume:         response.BaseVolume,
		OpenTime:           response.OpenTime,
		CloseTime:          response.CloseTime,
		FirstID:            response.FirstID,
		LastID:             response.LastID,
		Count:              response.Count,
	}, nil
}

func (c *Client) CoinMFuturesTickerPrice(ctx context.Context, symbol string) (exchange.COINMFuturesPriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.COINMFuturesPriceTicker{}, err
	}
	var responses []coinMFuturesPriceTickerResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/ticker/price", url.Values{"symbol": []string{normalized}}, &responses); err != nil {
		return exchange.COINMFuturesPriceTicker{}, err
	}
	if len(responses) == 0 {
		return exchange.COINMFuturesPriceTicker{}, fmt.Errorf("decode COIN-M Futures price ticker response: empty response")
	}
	response := responses[0]
	return exchange.COINMFuturesPriceTicker{Symbol: response.Symbol, Pair: response.Pair, Price: response.Price, Time: response.Time}, nil
}

func (c *Client) CoinMFuturesBookTicker(ctx context.Context, symbol string) (exchange.COINMFuturesBookTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.COINMFuturesBookTicker{}, err
	}
	var responses []coinMFuturesBookTickerResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/ticker/bookTicker", url.Values{"symbol": []string{normalized}}, &responses); err != nil {
		return exchange.COINMFuturesBookTicker{}, err
	}
	if len(responses) == 0 {
		return exchange.COINMFuturesBookTicker{}, fmt.Errorf("decode COIN-M Futures book ticker response: empty response")
	}
	response := responses[0]
	return exchange.COINMFuturesBookTicker{
		LastUpdateID: response.LastUpdateID,
		Symbol:       response.Symbol,
		Pair:         response.Pair,
		BidPrice:     response.BidPrice,
		BidQty:       response.BidQty,
		AskPrice:     response.AskPrice,
		AskQty:       response.AskQty,
		Time:         response.Time,
	}, nil
}

func (c *Client) CoinMFuturesPremiumIndex(ctx context.Context, symbol string) (exchange.COINMFuturesPremiumIndex, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.COINMFuturesPremiumIndex{}, err
	}
	var responses []coinMFuturesPremiumIndexResponse
	if err := c.getCoinMFuturesJSON(ctx, "/dapi/v1/premiumIndex", url.Values{"symbol": []string{normalized}}, &responses); err != nil {
		return exchange.COINMFuturesPremiumIndex{}, err
	}
	if len(responses) == 0 {
		return exchange.COINMFuturesPremiumIndex{}, fmt.Errorf("decode COIN-M Futures premium index response: empty response")
	}
	response := responses[0]
	return exchange.COINMFuturesPremiumIndex{
		Symbol:               response.Symbol,
		Pair:                 response.Pair,
		MarkPrice:            response.MarkPrice,
		IndexPrice:           response.IndexPrice,
		EstimatedSettlePrice: response.EstimatedSettlePrice,
		LastFundingRate:      response.LastFundingRate,
		InterestRate:         response.InterestRate,
		NextFundingTime:      response.NextFundingTime,
		Time:                 response.Time,
	}, nil
}
