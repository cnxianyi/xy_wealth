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
	maxFuturesDepthLimit      = 1000
	maxFuturesKlinesLimit     = 1500
	defaultFuturesDepthLimit  = 100
	defaultFuturesKlinesLimit = 500
)

var validFuturesIntervals = map[string]struct{}{
	"1m": {}, "3m": {}, "5m": {}, "15m": {}, "30m": {},
	"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
	"1d": {}, "3d": {}, "1w": {}, "1M": {},
}

type futuresExchangeInfoResponse struct {
	Timezone        string                      `json:"timezone"`
	ServerTime      int64                       `json:"serverTime"`
	FuturesType     string                      `json:"futuresType"`
	RateLimits      []rateLimitResponse         `json:"rateLimits"`
	ExchangeFilters []map[string]any            `json:"exchangeFilters"`
	Assets          []futuresAssetResponse      `json:"assets"`
	Symbols         []futuresSymbolInfoResponse `json:"symbols"`
}

type futuresAssetResponse struct {
	Asset             string `json:"asset"`
	MarginAvailable   bool   `json:"marginAvailable"`
	AutoAssetExchange string `json:"autoAssetExchange"`
}

type futuresSymbolInfoResponse struct {
	Symbol                string           `json:"symbol"`
	Pair                  string           `json:"pair"`
	ContractType          string           `json:"contractType"`
	DeliveryDate          int64            `json:"deliveryDate"`
	OnboardDate           int64            `json:"onboardDate"`
	Status                string           `json:"status"`
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
	SettlePlan            int              `json:"settlePlan"`
	TriggerProtect        string           `json:"triggerProtect"`
	LiquidationFee        string           `json:"liquidationFee"`
	MarketTakeBound       string           `json:"marketTakeBound"`
	MaxMoveOrderLimit     int              `json:"maxMoveOrderLimit"`
	OrderTypes            []string         `json:"orderTypes"`
	TimeInForce           []string         `json:"timeInForce"`
	PermissionSets        []string         `json:"permissionSets"`
	Filters               []map[string]any `json:"filters"`
}

type futuresOrderBookResponse struct {
	LastUpdateID    int64      `json:"lastUpdateId"`
	Symbol          string     `json:"symbol"`
	Pair            string     `json:"pair"`
	EventTime       int64      `json:"E"`
	TransactionTime int64      `json:"T"`
	Bids            [][]string `json:"bids"`
	Asks            [][]string `json:"asks"`
}

type futuresTicker24hrResponse struct {
	Symbol             string `json:"symbol"`
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
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	FirstID            int64  `json:"firstId"`
	LastID             int64  `json:"lastId"`
	Count              int64  `json:"count"`
}

type futuresPremiumIndexResponse struct {
	Symbol               string `json:"symbol"`
	MarkPrice            string `json:"markPrice"`
	IndexPrice           string `json:"indexPrice"`
	EstimatedSettlePrice string `json:"estimatedSettlePrice"`
	LastFundingRate      string `json:"lastFundingRate"`
	InterestRate         string `json:"interestRate"`
	NextFundingTime      int64  `json:"nextFundingTime"`
	Time                 int64  `json:"time"`
}

func (c *Client) FuturesPing(ctx context.Context) error {
	return c.getFuturesJSON(ctx, "/fapi/v1/ping", nil, nil)
}

func (c *Client) FuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	return exchange.ServerTime{ServerTime: response.ServerTime}, nil
}

func (c *Client) FuturesExchangeInfo(ctx context.Context) (exchange.USDSMFuturesExchangeInfo, error) {
	var response futuresExchangeInfoResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/exchangeInfo", nil, &response); err != nil {
		return exchange.USDSMFuturesExchangeInfo{}, err
	}

	info := exchange.USDSMFuturesExchangeInfo{
		Timezone:        response.Timezone,
		ServerTime:      response.ServerTime,
		FuturesType:     response.FuturesType,
		ExchangeFilters: response.ExchangeFilters,
		Assets:          make([]exchange.FuturesAsset, 0, len(response.Assets)),
		Symbols:         make([]exchange.USDSMFuturesSymbolInfo, 0, len(response.Symbols)),
		RateLimits:      make([]exchange.RateLimit, 0, len(response.RateLimits)),
	}
	for _, limit := range response.RateLimits {
		info.RateLimits = append(info.RateLimits, exchange.RateLimit{
			RateLimitType: limit.RateLimitType,
			Interval:      limit.Interval,
			IntervalNum:   limit.IntervalNum,
			Limit:         limit.Limit,
		})
	}
	for _, assetInfo := range response.Assets {
		info.Assets = append(info.Assets, exchange.FuturesAsset{
			Asset:             assetInfo.Asset,
			MarginAvailable:   assetInfo.MarginAvailable,
			AutoAssetExchange: assetInfo.AutoAssetExchange,
		})
	}
	for _, symbolInfo := range response.Symbols {
		info.Symbols = append(info.Symbols, exchange.USDSMFuturesSymbolInfo{
			Symbol:                symbolInfo.Symbol,
			Pair:                  symbolInfo.Pair,
			ContractType:          symbolInfo.ContractType,
			DeliveryDate:          symbolInfo.DeliveryDate,
			OnboardDate:           symbolInfo.OnboardDate,
			Status:                symbolInfo.Status,
			MaintMarginPercent:    symbolInfo.MaintMarginPercent,
			RequiredMarginPercent: symbolInfo.RequiredMarginPercent,
			BaseAsset:             symbolInfo.BaseAsset,
			QuoteAsset:            symbolInfo.QuoteAsset,
			MarginAsset:           symbolInfo.MarginAsset,
			PricePrecision:        symbolInfo.PricePrecision,
			QuantityPrecision:     symbolInfo.QuantityPrecision,
			BaseAssetPrecision:    symbolInfo.BaseAssetPrecision,
			QuotePrecision:        symbolInfo.QuotePrecision,
			UnderlyingType:        symbolInfo.UnderlyingType,
			UnderlyingSubType:     symbolInfo.UnderlyingSubType,
			SettlePlan:            symbolInfo.SettlePlan,
			TriggerProtect:        symbolInfo.TriggerProtect,
			LiquidationFee:        symbolInfo.LiquidationFee,
			MarketTakeBound:       symbolInfo.MarketTakeBound,
			MaxMoveOrderLimit:     symbolInfo.MaxMoveOrderLimit,
			OrderTypes:            symbolInfo.OrderTypes,
			TimeInForce:           symbolInfo.TimeInForce,
			PermissionSets:        symbolInfo.PermissionSets,
			Filters:               symbolInfo.Filters,
		})
	}
	return info, nil
}

func (c *Client) FuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	if limit == 0 {
		limit = defaultFuturesDepthLimit
	}
	if err := validateLimit("limit", limit, maxFuturesDepthLimit); err != nil {
		return exchange.FuturesOrderBook{}, err
	}

	query := url.Values{"symbol": []string{normalized}, "limit": []string{strconv.Itoa(limit)}}
	var response futuresOrderBookResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/depth", query, &response); err != nil {
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

func (c *Client) FuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(request.Interval)
	if _, ok := validFuturesIntervals[interval]; !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported USDⓈ-M Futures interval"}
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultFuturesKlinesLimit
	}
	if err := validateLimit("limit", limit, maxFuturesKlinesLimit); err != nil {
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
	if timezone := strings.TrimSpace(request.TimeZone); timezone != "" {
		query.Set("timeZone", timezone)
	}

	var raw []json.RawMessage
	if err := c.getFuturesJSON(ctx, "/fapi/v1/klines", query, &raw); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(raw))
	for index, item := range raw {
		kline, err := decodeKline(item)
		if err != nil {
			return nil, fmt.Errorf("decode futures kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	return klines, nil
}

func (c *Client) FuturesTicker24hr(ctx context.Context, symbol string) (exchange.FuturesTicker24hr, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	var response futuresTicker24hrResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/ticker/24hr", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	return exchange.FuturesTicker24hr{
		Symbol:             response.Symbol,
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
		QuoteVolume:        response.QuoteVolume,
		OpenTime:           response.OpenTime,
		CloseTime:          response.CloseTime,
		FirstID:            response.FirstID,
		LastID:             response.LastID,
		Count:              response.Count,
	}, nil
}

func (c *Client) FuturesTickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	var response priceTickerResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/ticker/price", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.PriceTicker{}, err
	}
	return exchange.PriceTicker{Symbol: response.Symbol, Price: response.Price}, nil
}

func (c *Client) FuturesBookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	var response bookTickerResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/ticker/bookTicker", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.BookTicker{}, err
	}
	return exchange.BookTicker{
		Symbol:   response.Symbol,
		BidPrice: response.BidPrice,
		BidQty:   response.BidQty,
		AskPrice: response.AskPrice,
		AskQty:   response.AskQty,
	}, nil
}

func (c *Client) FuturesPremiumIndex(ctx context.Context, symbol string) (exchange.FuturesPremiumIndex, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesPremiumIndex{}, err
	}
	var response futuresPremiumIndexResponse
	if err := c.getFuturesJSON(ctx, "/fapi/v1/premiumIndex", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.FuturesPremiumIndex{}, err
	}
	return exchange.FuturesPremiumIndex{
		Symbol:               response.Symbol,
		MarkPrice:            response.MarkPrice,
		IndexPrice:           response.IndexPrice,
		EstimatedSettlePrice: response.EstimatedSettlePrice,
		LastFundingRate:      response.LastFundingRate,
		InterestRate:         response.InterestRate,
		NextFundingTime:      response.NextFundingTime,
		Time:                 response.Time,
	}, nil
}
