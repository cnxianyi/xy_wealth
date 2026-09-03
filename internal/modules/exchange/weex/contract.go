package weex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

const (
	weexContractPathPrefix     = "/capi/v3"
	defaultContractDepthLimit  = 15
	defaultContractKlinesLimit = 100
	maxContractKlinesLimit     = 1000
)

var validContractIntervals = map[string]struct{}{
	"1m": {}, "5m": {}, "15m": {}, "30m": {},
	"1h": {}, "4h": {}, "12h": {}, "1d": {}, "1w": {},
}

type contractExchangeInfoResponse struct {
	Assets     []contractAssetResponse  `json:"assets"`
	RateLimits []rateLimitResponse      `json:"rateLimits"`
	Symbols    []contractSymbolResponse `json:"symbols"`
}

type contractAssetResponse struct {
	Asset           string `json:"asset"`
	MarginAvailable bool   `json:"marginAvailable"`
}

type contractSymbolResponse struct {
	Symbol              string       `json:"symbol"`
	DisplaySymbol       string       `json:"displaySymbol"`
	BaseAsset           string       `json:"baseAsset"`
	QuoteAsset          string       `json:"quoteAsset"`
	MarginAsset         string       `json:"marginAsset"`
	ContractType        string       `json:"contractType"`
	UnderlyingType      string       `json:"underlyingType"`
	UnderlyingSubType   []string     `json:"underlyingSubType"`
	PricePrecision      int          `json:"pricePrecision"`
	QuantityPrecision   int          `json:"quantityPrecision"`
	BaseAssetPrecision  int          `json:"baseAssetPrecision"`
	QuotePrecision      int          `json:"quotePrecision"`
	ContractVal         numberString `json:"contractVal"`
	Delivery            []string     `json:"delivery"`
	ForwardContractFlag bool         `json:"forwardContractFlag"`
	MinLeverage         int          `json:"minLeverage"`
	MaxLeverage         int          `json:"maxLeverage"`
	BuyLimitPriceRatio  numberString `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio numberString `json:"sellLimitPriceRatio"`
	MakerFeeRate        numberString `json:"makerFeeRate"`
	TakerFeeRate        numberString `json:"takerFeeRate"`
	APIMakerFeeRate     numberString `json:"apiMakerFeeRate"`
	APITakerFeeRate     numberString `json:"apiTakerFeeRate"`
	MinOrderSize        numberString `json:"minOrderSize"`
	MaxOrderSize        numberString `json:"maxOrderSize"`
	MaxPositionSize     numberString `json:"maxPositionSize"`
	MarketOpenLimitSize numberString `json:"marketOpenLimitSize"`
}

type contractTicker24hrResponse struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	MarkPrice          string `json:"markPrice"`
	IndexPrice         string `json:"indexPrice"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
}

type contractPriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
	Time   int64  `json:"time"`
}

type contractBookTickerResponse struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	BidQty   string `json:"bidQty"`
	AskPrice string `json:"askPrice"`
	AskQty   string `json:"askQty"`
	Time     int64  `json:"time"`
}

type contractPremiumIndexResponse struct {
	Symbol              string `json:"symbol"`
	MarkPrice           string `json:"markPrice"`
	IndexPrice          string `json:"indexPrice"`
	LastFundingRate     string `json:"lastFundingRate"`
	ForecastFundingRate string `json:"forecastFundingRate"`
	InterestRate        string `json:"interestRate"`
	NextFundingTime     int64  `json:"nextFundingTime"`
	Time                int64  `json:"time"`
	CollectCycle        int64  `json:"collectCycle"`
}

type contractBalanceResponse struct {
	Asset            string `json:"asset"`
	Balance          string `json:"balance"`
	AvailableBalance string `json:"availableBalance"`
	Frozen           string `json:"frozen"`
	UnrealizePnl     string `json:"unrealizePnl"`
}

type contractPositionResponse struct {
	ID                         int64  `json:"id"`
	Asset                      string `json:"asset"`
	Symbol                     string `json:"symbol"`
	Side                       string `json:"side"`
	MarginType                 string `json:"marginType"`
	SeparatedMode              string `json:"separatedMode"`
	SeparatedOpenOrderID       int64  `json:"separatedOpenOrderId"`
	Leverage                   string `json:"leverage"`
	Size                       string `json:"size"`
	OpenValue                  string `json:"openValue"`
	OpenFee                    string `json:"openFee"`
	FundingFee                 string `json:"fundingFee"`
	MarginSize                 string `json:"marginSize"`
	IsolatedMargin             string `json:"isolatedMargin"`
	IsAutoAppendIsolatedMargin bool   `json:"isAutoAppendIsolatedMargin"`
	CumOpenSize                string `json:"cumOpenSize"`
	CumOpenValue               string `json:"cumOpenValue"`
	CumOpenFee                 string `json:"cumOpenFee"`
	CumCloseSize               string `json:"cumCloseSize"`
	CumCloseValue              string `json:"cumCloseValue"`
	CumCloseFee                string `json:"cumCloseFee"`
	CumFundingFee              string `json:"cumFundingFee"`
	CumLiquidateFee            string `json:"cumLiquidateFee"`
	CreatedMatchSequenceID     int64  `json:"createdMatchSequenceId"`
	UpdatedMatchSequenceID     int64  `json:"updatedMatchSequenceId"`
	CreatedTime                int64  `json:"createdTime"`
	UpdatedTime                int64  `json:"updatedTime"`
	UnrealizePnl               string `json:"unrealizePnl"`
	LiquidatePrice             string `json:"liquidatePrice"`
}

var _ exchange.USDSMFuturesProvider = (*Client)(nil)
var _ exchange.ContractPositionProvider = (*Client)(nil)

// FuturesPing checks that the Weex Contract market API is reachable. Weex
// Contract V3 does not expose a dedicated /ping endpoint, so the public
// server-time endpoint is used as the lightweight connectivity check.
func (c *Client) FuturesPing(ctx context.Context) error {
	return c.getContractJSON(ctx, weexContractPathPrefix+"/market/time", nil, nil)
}

// FuturesServerTime returns Weex Contract's server time in milliseconds.
func (c *Client) FuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	return exchange.ServerTime{ServerTime: response.ServerTime}, nil
}

// FuturesExchangeInfo returns Weex Contract trading rules and collateral
// assets. Weex does not return Binance's timezone or status fields, so the
// shared model uses UTC and TRADING for those stable contract metadata values.
func (c *Client) FuturesExchangeInfo(ctx context.Context) (exchange.USDSMFuturesExchangeInfo, error) {
	var response contractExchangeInfoResponse
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/exchangeInfo", nil, &response); err != nil {
		return exchange.USDSMFuturesExchangeInfo{}, err
	}

	info := exchange.USDSMFuturesExchangeInfo{
		Timezone:    "UTC",
		FuturesType: "WEEX_CONTRACT",
		Assets:      make([]exchange.FuturesAsset, 0, len(response.Assets)),
		Symbols:     make([]exchange.USDSMFuturesSymbolInfo, 0, len(response.Symbols)),
		RateLimits:  make([]exchange.RateLimit, 0, len(response.RateLimits)),
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
			Asset:           assetInfo.Asset,
			MarginAvailable: assetInfo.MarginAvailable,
		})
	}
	for _, symbolInfo := range response.Symbols {
		info.Symbols = append(info.Symbols, exchange.USDSMFuturesSymbolInfo{
			Symbol:              symbolInfo.Symbol,
			Pair:                symbolInfo.DisplaySymbol,
			ContractType:        symbolInfo.ContractType,
			DisplaySymbol:       symbolInfo.DisplaySymbol,
			Status:              "TRADING",
			BaseAsset:           symbolInfo.BaseAsset,
			QuoteAsset:          symbolInfo.QuoteAsset,
			MarginAsset:         symbolInfo.MarginAsset,
			PricePrecision:      symbolInfo.PricePrecision,
			QuantityPrecision:   symbolInfo.QuantityPrecision,
			BaseAssetPrecision:  symbolInfo.BaseAssetPrecision,
			QuotePrecision:      symbolInfo.QuotePrecision,
			UnderlyingType:      symbolInfo.UnderlyingType,
			UnderlyingSubType:   symbolInfo.UnderlyingSubType,
			ContractVal:         string(symbolInfo.ContractVal),
			Delivery:            symbolInfo.Delivery,
			ForwardContractFlag: symbolInfo.ForwardContractFlag,
			MinLeverage:         symbolInfo.MinLeverage,
			MaxLeverage:         symbolInfo.MaxLeverage,
			BuyLimitPriceRatio:  string(symbolInfo.BuyLimitPriceRatio),
			SellLimitPriceRatio: string(symbolInfo.SellLimitPriceRatio),
			MakerFeeRate:        string(symbolInfo.MakerFeeRate),
			TakerFeeRate:        string(symbolInfo.TakerFeeRate),
			APIMakerFeeRate:     string(symbolInfo.APIMakerFeeRate),
			APITakerFeeRate:     string(symbolInfo.APITakerFeeRate),
			MinOrderSize:        string(symbolInfo.MinOrderSize),
			MaxOrderSize:        string(symbolInfo.MaxOrderSize),
			MaxPositionSize:     string(symbolInfo.MaxPositionSize),
			MarketOpenLimitSize: string(symbolInfo.MarketOpenLimitSize),
		})
	}
	return info, nil
}

// FuturesDepth returns the current Weex Contract order book.
func (c *Client) FuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	if limit == 0 {
		limit = defaultContractDepthLimit
	}
	if _, ok := validDepthLimits[limit]; !ok {
		return exchange.FuturesOrderBook{}, exchange.InvalidParameterError{
			Parameter: "limit",
			Message:   "must be one of 15 or 200",
		}
	}

	query := url.Values{"symbol": []string{normalized}, "limit": []string{strconv.Itoa(limit)}}
	var response orderBookResponse
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/depth", query, &response); err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	return exchange.FuturesOrderBook{
		LastUpdateID: response.LastUpdateID,
		Symbol:       normalized,
		Bids:         response.Bids,
		Asks:         response.Asks,
	}, nil
}

// FuturesKlines returns Weex Contract candlesticks. The upstream response has
// the same 11-field array shape as Spot, so the shared decoder is reused. The
// generic timeZone field is intentionally omitted because Weex does not
// support it.
func (c *Client) FuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(request.Interval)
	if _, ok := validContractIntervals[interval]; !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported Weex Contract interval"}
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultContractKlinesLimit
	}
	maxLimit := maxContractKlinesLimit
	path := weexContractPathPrefix + "/market/klines"
	if request.StartTime != nil || request.EndTime != nil {
		// The current-window endpoint ignores time bounds. Use the documented
		// history endpoint whenever a caller supplies either bound.
		path = weexContractPathPrefix + "/market/historyKlines"
		maxLimit = 100
	}
	if err := validateLimit("limit", limit, maxLimit); err != nil {
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
	if err := c.getContractJSON(ctx, path, query, &raw); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(raw))
	for index, item := range raw {
		kline, err := decodeKline(item)
		if err != nil {
			return nil, fmt.Errorf("decode contract kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	if len(klines) > limit {
		klines = klines[:limit]
	}
	return klines, nil
}

// FuturesTicker24hr returns Weex Contract's 24-hour statistics. Weex does not
// return a weighted average, so it is calculated from quoteVolume / volume.
func (c *Client) FuturesTicker24hr(ctx context.Context, symbol string) (exchange.FuturesTicker24hr, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	var response responseList[contractTicker24hrResponse]
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/ticker/24hr", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	if len(response) == 0 {
		return exchange.FuturesTicker24hr{}, errors.New("weex contract 24-hour ticker response is empty")
	}
	item := response[0]
	return exchange.FuturesTicker24hr{
		Symbol:             item.Symbol,
		PriceChange:        item.PriceChange,
		PriceChangePercent: item.PriceChangePercent,
		WeightedAvgPrice:   calculateWeightedAvgPrice(item.Volume, item.QuoteVolume),
		LastPrice:          item.LastPrice,
		OpenPrice:          item.OpenPrice,
		HighPrice:          item.HighPrice,
		LowPrice:           item.LowPrice,
		Volume:             item.Volume,
		QuoteVolume:        item.QuoteVolume,
		MarkPrice:          item.MarkPrice,
		IndexPrice:         item.IndexPrice,
		OpenTime:           item.OpenTime,
		CloseTime:          item.CloseTime,
	}, nil
}

// FuturesTickerPrice returns the default INDEX price for a contract. Weex's
// V3 endpoint also accepts priceType=MARK, but the shared interface does not
// expose a price-type argument yet.
func (c *Client) FuturesTickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	var response contractPriceResponse
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/symbolPrice", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.PriceTicker{}, err
	}
	return exchange.PriceTicker{Symbol: response.Symbol, Price: response.Price, Time: response.Time}, nil
}

// FuturesBookTicker returns the best bid and ask for a contract.
func (c *Client) FuturesBookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	var response responseList[contractBookTickerResponse]
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/ticker/bookTicker", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.BookTicker{}, err
	}
	if len(response) == 0 {
		return exchange.BookTicker{}, errors.New("weex contract book ticker response is empty")
	}
	item := response[0]
	return exchange.BookTicker{
		Symbol:   item.Symbol,
		BidPrice: item.BidPrice,
		BidQty:   item.BidQty,
		AskPrice: item.AskPrice,
		AskQty:   item.AskQty,
		Time:     item.Time,
	}, nil
}

// FuturesPremiumIndex returns the latest mark/index prices and funding data.
func (c *Client) FuturesPremiumIndex(ctx context.Context, symbol string) (exchange.FuturesPremiumIndex, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesPremiumIndex{}, err
	}
	var response responseList[contractPremiumIndexResponse]
	if err := c.getContractJSON(ctx, weexContractPathPrefix+"/market/premiumIndex", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.FuturesPremiumIndex{}, err
	}
	if len(response) == 0 {
		return exchange.FuturesPremiumIndex{}, errors.New("weex contract premium index response is empty")
	}
	item := response[0]
	return exchange.FuturesPremiumIndex{
		Symbol:              item.Symbol,
		MarkPrice:           item.MarkPrice,
		IndexPrice:          item.IndexPrice,
		LastFundingRate:     item.LastFundingRate,
		ForecastFundingRate: item.ForecastFundingRate,
		InterestRate:        item.InterestRate,
		NextFundingTime:     item.NextFundingTime,
		Time:                item.Time,
		CollectCycle:        item.CollectCycle,
	}, nil
}

// ContractBalances calls Weex Contract's signed balance endpoint. The generic
// Provider.Balances method continues to represent Spot balances; this method
// is exposed for the future contract-account capability without changing that
// existing API contract.
func (c *Client) ContractBalances(ctx context.Context) ([]asset.Balance, error) {
	var response []contractBalanceResponse
	if err := c.getSignedContractJSON(ctx, weexContractPathPrefix+"/account/balance", nil, &response); err != nil {
		return nil, err
	}

	balances := make([]asset.Balance, 0, len(response))
	for _, item := range response {
		balance, err := decimal.NewFromString(item.Balance)
		if err != nil {
			return nil, fmt.Errorf("parse contract %s balance: %w", item.Asset, err)
		}
		available, err := decimal.NewFromString(item.AvailableBalance)
		if err != nil {
			return nil, fmt.Errorf("parse contract %s available balance: %w", item.Asset, err)
		}
		frozen, err := decimal.NewFromString(item.Frozen)
		if err != nil {
			return nil, fmt.Errorf("parse contract %s frozen balance: %w", item.Asset, err)
		}
		if !c.includeZero && balance.IsZero() && available.IsZero() && frozen.IsZero() {
			continue
		}
		balances = append(balances, asset.Balance{
			Symbol: item.Asset,
			Free:   available.String(),
			Locked: frozen.String(),
			Total:  balance.String(),
		})
	}
	return balances, nil
}

// ContractPositions returns all open Weex Contract positions. Supplying a
// symbol uses Weex's single-position endpoint; an empty symbol queries all
// positions.
func (c *Client) ContractPositions(ctx context.Context, symbol string) ([]exchange.ContractPosition, error) {
	path := weexContractPathPrefix + "/account/position/allPosition"
	query := url.Values{}
	if strings.TrimSpace(symbol) != "" {
		normalized, err := normalizeSymbol(symbol)
		if err != nil {
			return nil, err
		}
		path = weexContractPathPrefix + "/account/position/singlePosition"
		query.Set("symbol", normalized)
	}

	var response []contractPositionResponse
	if err := c.getSignedContractJSON(ctx, path, query, &response); err != nil {
		return nil, err
	}
	positions := make([]exchange.ContractPosition, 0, len(response))
	for _, item := range response {
		if !c.includeZero {
			size := strings.TrimSpace(item.Size)
			if size != "" {
				amount, err := decimal.NewFromString(size)
				if err != nil {
					return nil, fmt.Errorf("parse contract %s position size: %w", item.Symbol, err)
				}
				if amount.IsZero() {
					continue
				}
			}
		}
		positions = append(positions, exchange.ContractPosition{
			ID:                         item.ID,
			Asset:                      item.Asset,
			Symbol:                     item.Symbol,
			Side:                       item.Side,
			MarginType:                 item.MarginType,
			SeparatedMode:              item.SeparatedMode,
			SeparatedOpenOrderID:       item.SeparatedOpenOrderID,
			Leverage:                   item.Leverage,
			Size:                       item.Size,
			OpenValue:                  item.OpenValue,
			OpenFee:                    item.OpenFee,
			FundingFee:                 item.FundingFee,
			MarginSize:                 item.MarginSize,
			IsolatedMargin:             item.IsolatedMargin,
			IsAutoAppendIsolatedMargin: item.IsAutoAppendIsolatedMargin,
			CumOpenSize:                item.CumOpenSize,
			CumOpenValue:               item.CumOpenValue,
			CumOpenFee:                 item.CumOpenFee,
			CumCloseSize:               item.CumCloseSize,
			CumCloseValue:              item.CumCloseValue,
			CumCloseFee:                item.CumCloseFee,
			CumFundingFee:              item.CumFundingFee,
			CumLiquidateFee:            item.CumLiquidateFee,
			CreatedMatchSequenceID:     item.CreatedMatchSequenceID,
			UpdatedMatchSequenceID:     item.UpdatedMatchSequenceID,
			CreatedTime:                item.CreatedTime,
			UpdatedTime:                item.UpdatedTime,
			UnrealizePnl:               item.UnrealizePnl,
			LiquidatePrice:             item.LiquidatePrice,
		})
	}
	return positions, nil
}
