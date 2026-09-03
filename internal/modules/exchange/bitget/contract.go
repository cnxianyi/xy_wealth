package bitget

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

const bitgetUSDTFuturesProductType = "USDT-FUTURES"

var validContractIntervals = map[string]string{
	"1m": "1m", "3m": "3m", "5m": "5m", "15m": "15m", "30m": "30m",
	"1h": "1H", "4h": "4H", "6h": "6H", "12h": "12H",
	"1d": "1D", "3d": "3D", "1w": "1W", "1M": "1M",
}

type contractSymbolResponse struct {
	Symbol              string       `json:"symbol"`
	BaseCoin            string       `json:"baseCoin"`
	QuoteCoin           string       `json:"quoteCoin"`
	BuyLimitPriceRatio  numberString `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio numberString `json:"sellLimitPriceRatio"`
	MakerFeeRate        numberString `json:"makerFeeRate"`
	TakerFeeRate        numberString `json:"takerFeeRate"`
	SupportMarginCoins  []string     `json:"supportMarginCoins"`
	MinTradeNum         numberString `json:"minTradeNum"`
	VolumePlace         intValue     `json:"volumePlace"`
	PricePlace          intValue     `json:"pricePlace"`
	SizeMultiplier      numberString `json:"sizeMultiplier"`
	MinTradeUSDT        numberString `json:"minTradeUSDT"`
	MaxSymbolOrderNum   intValue     `json:"maxSymbolOrderNum"`
	MaxPositionNum      intValue     `json:"maxPositionNum"`
	SymbolType          string       `json:"symbolType"`
	SymbolStatus        string       `json:"symbolStatus"`
	DeliveryTime        numberString `json:"deliveryTime"`
	LaunchTime          numberString `json:"launchTime"`
	MinLever            intValue     `json:"minLever"`
	MaxLever            intValue     `json:"maxLever"`
	MaxMarketOrderQty   numberString `json:"maxMarketOrderQty"`
	MaxOrderQty         numberString `json:"maxOrderQty"`
}

type contractTickerResponse struct {
	Symbol      string       `json:"symbol"`
	LastPrice   numberString `json:"lastPr"`
	AskPrice    numberString `json:"askPr"`
	BidPrice    numberString `json:"bidPr"`
	BidQty      numberString `json:"bidSz"`
	AskQty      numberString `json:"askSz"`
	High24h     numberString `json:"high24h"`
	Low24h      numberString `json:"low24h"`
	Time        numberString `json:"ts"`
	Change24h   numberString `json:"change24h"`
	BaseVolume  numberString `json:"baseVolume"`
	QuoteVolume numberString `json:"quoteVolume"`
	Open24h     numberString `json:"open24h"`
	IndexPrice  numberString `json:"indexPrice"`
	FundingRate numberString `json:"fundingRate"`
	MarkPrice   numberString `json:"markPrice"`
}

type contractPriceResponse struct {
	Symbol     string       `json:"symbol"`
	Price      numberString `json:"price"`
	IndexPrice numberString `json:"indexPrice"`
	MarkPrice  numberString `json:"markPrice"`
	Time       numberString `json:"ts"`
}

type contractDepthResponse struct {
	Asks [][]json.RawMessage `json:"asks"`
	Bids [][]json.RawMessage `json:"bids"`
	Time numberString        `json:"ts"`
}

var _ exchange.USDSMFuturesProvider = (*Client)(nil)

// FuturesPing tests Bitget's public API. Bitget has no product-specific ping
// endpoint, so the public server-time endpoint is used as the health probe.
func (c *Client) FuturesPing(ctx context.Context) error {
	var response serverTimeResponse
	return c.getJSON(ctx, "/public/time", nil, &response)
}

// FuturesServerTime returns Bitget's public server time for the futures API.
func (c *Client) FuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	return c.ServerTime(ctx)
}

// FuturesExchangeInfo returns Bitget USDT-FUTURES contract configuration.
func (c *Client) FuturesExchangeInfo(ctx context.Context) (exchange.USDSMFuturesExchangeInfo, error) {
	response, err := c.contracts(ctx, bitgetUSDTFuturesProductType)
	if err != nil {
		return exchange.USDSMFuturesExchangeInfo{}, err
	}
	return normalizeUSDTFuturesExchangeInfo(response)
}

func (c *Client) contracts(ctx context.Context, productType string) ([]contractSymbolResponse, error) {
	query := url.Values{"productType": []string{productType}}
	var response []contractSymbolResponse
	if err := c.getJSON(ctx, "/mix/market/contracts", query, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func normalizeUSDTFuturesExchangeInfo(response []contractSymbolResponse) (exchange.USDSMFuturesExchangeInfo, error) {
	return normalizeFuturesExchangeInfo(response, bitgetUSDTFuturesProductType)
}

func normalizeFuturesExchangeInfo(response []contractSymbolResponse, productType string) (exchange.USDSMFuturesExchangeInfo, error) {
	info := exchange.USDSMFuturesExchangeInfo{
		Timezone:    "UTC",
		FuturesType: productType,
		Symbols:     make([]exchange.USDSMFuturesSymbolInfo, 0, len(response)),
	}
	for _, item := range response {
		marginAsset := item.QuoteCoin
		if len(item.SupportMarginCoins) > 0 {
			marginAsset = item.SupportMarginCoins[0]
		}
		deliveryDate, err := parseOptionalInt(string(item.DeliveryTime))
		if err != nil {
			return exchange.USDSMFuturesExchangeInfo{}, fmt.Errorf("parse %s delivery time: %w", item.Symbol, err)
		}
		onboardDate, err := parseOptionalInt(string(item.LaunchTime))
		if err != nil {
			return exchange.USDSMFuturesExchangeInfo{}, fmt.Errorf("parse %s launch time: %w", item.Symbol, err)
		}
		info.Symbols = append(info.Symbols, exchange.USDSMFuturesSymbolInfo{
			Symbol:              item.Symbol,
			Pair:                item.Symbol,
			ContractType:        strings.ToUpper(item.SymbolType),
			DeliveryDate:        deliveryDate,
			OnboardDate:         onboardDate,
			Status:              normalizeContractStatus(item.SymbolStatus),
			BaseAsset:           item.BaseCoin,
			QuoteAsset:          item.QuoteCoin,
			MarginAsset:         marginAsset,
			PricePrecision:      int(item.PricePlace),
			QuantityPrecision:   int(item.VolumePlace),
			BaseAssetPrecision:  int(item.VolumePlace),
			QuotePrecision:      int(item.PricePlace),
			ContractVal:         string(item.SizeMultiplier),
			MinLeverage:         int(item.MinLever),
			MaxLeverage:         int(item.MaxLever),
			BuyLimitPriceRatio:  string(item.BuyLimitPriceRatio),
			SellLimitPriceRatio: string(item.SellLimitPriceRatio),
			MinTradeUSDT:        string(item.MinTradeUSDT),
			MakerFeeRate:        string(item.MakerFeeRate),
			TakerFeeRate:        string(item.TakerFeeRate),
			MinOrderSize:        string(item.MinTradeNum),
			MaxOrderSize:        string(item.MaxOrderQty),
			MaxPositionSize:     strconv.Itoa(int(item.MaxPositionNum)),
			MarketOpenLimitSize: string(item.MaxMarketOrderQty),
		})
	}
	return info, nil
}

// FuturesDepth returns Bitget's merged USDT-FUTURES order book. Bitget sends
// price and quantity as JSON numbers for this endpoint, so levels are decoded
// explicitly into decimal strings before crossing the module boundary.
func (c *Client) FuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	return c.futuresDepth(ctx, symbol, limit, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresDepth(ctx context.Context, symbol string, limit int, productType string) (exchange.FuturesOrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	if limit == 0 {
		limit = 100
	}
	if err := validateContractDepthLimit(limit); err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	query := url.Values{
		"productType": []string{productType},
		"symbol":      []string{normalized},
		"limit":       []string{strconv.Itoa(limit)},
	}
	var response contractDepthResponse
	if err := c.getJSON(ctx, "/mix/market/merge-depth", query, &response); err != nil {
		return exchange.FuturesOrderBook{}, err
	}
	bids, err := decodeLevels(response.Bids)
	if err != nil {
		return exchange.FuturesOrderBook{}, fmt.Errorf("decode futures bids: %w", err)
	}
	asks, err := decodeLevels(response.Asks)
	if err != nil {
		return exchange.FuturesOrderBook{}, fmt.Errorf("decode futures asks: %w", err)
	}
	updateTime, err := parseOptionalInt(string(response.Time))
	if err != nil {
		return exchange.FuturesOrderBook{}, fmt.Errorf("parse futures depth timestamp: %w", err)
	}
	return exchange.FuturesOrderBook{
		Symbol:          normalized,
		LastUpdateID:    0,
		EventTime:       updateTime,
		TransactionTime: updateTime,
		Bids:            bids,
		Asks:            asks,
	}, nil
}

func (c *Client) FuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	return c.futuresKlines(ctx, request, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresKlines(ctx context.Context, request exchange.KlinesRequest, productType string) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	granularity, ok := validContractIntervals[strings.TrimSpace(request.Interval)]
	if !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported Bitget futures interval"}
	}
	limit := request.Limit
	if limit == 0 {
		limit = defaultKlinesLimit
	}
	if err := validateLimit("limit", limit, maxKlinesLimit); err != nil {
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
		"productType": []string{productType},
		"symbol":      []string{normalized},
		"granularity": []string{granularity},
		"limit":       []string{strconv.Itoa(limit)},
	}
	if request.StartTime != nil {
		query.Set("startTime", strconv.FormatInt(*request.StartTime, 10))
	}
	if request.EndTime != nil {
		query.Set("endTime", strconv.FormatInt(*request.EndTime, 10))
	}
	var response [][]json.RawMessage
	if err := c.getJSON(ctx, "/mix/market/candles", query, &response); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(response))
	for index, values := range response {
		kline, err := decodeKline(values)
		if err != nil {
			return nil, fmt.Errorf("decode Bitget futures kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	if len(klines) > limit {
		klines = klines[:limit]
	}
	return klines, nil
}

func (c *Client) FuturesTicker24hr(ctx context.Context, symbol string) (exchange.FuturesTicker24hr, error) {
	return c.futuresTicker24hr(ctx, symbol, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresTicker24hr(ctx context.Context, symbol, productType string) (exchange.FuturesTicker24hr, error) {
	item, err := c.contractTicker(ctx, symbol, productType)
	if err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	last, err := decimal.NewFromString(string(item.LastPrice))
	if err != nil {
		return exchange.FuturesTicker24hr{}, fmt.Errorf("parse futures ticker last price: %w", err)
	}
	open, err := decimal.NewFromString(string(item.Open24h))
	if err != nil {
		return exchange.FuturesTicker24hr{}, fmt.Errorf("parse futures ticker open price: %w", err)
	}
	priceChange := last.Sub(open)
	changePercent := ""
	if strings.TrimSpace(string(item.Change24h)) != "" {
		change, err := decimal.NewFromString(string(item.Change24h))
		if err != nil {
			return exchange.FuturesTicker24hr{}, fmt.Errorf("parse futures ticker 24-hour change: %w", err)
		}
		changePercent = change.Mul(decimal.NewFromInt(100)).String()
	} else if !open.IsZero() {
		changePercent = priceChange.Div(open).Mul(decimal.NewFromInt(100)).String()
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.FuturesTicker24hr{}, fmt.Errorf("parse futures ticker timestamp: %w", err)
	}
	weightedAverage, err := weightedAveragePrice(string(item.BaseVolume), string(item.QuoteVolume))
	if err != nil {
		return exchange.FuturesTicker24hr{}, err
	}
	return exchange.FuturesTicker24hr{
		Symbol:             item.Symbol,
		PriceChange:        priceChange.String(),
		PriceChangePercent: changePercent,
		WeightedAvgPrice:   weightedAverage,
		LastPrice:          string(item.LastPrice),
		OpenPrice:          string(item.Open24h),
		HighPrice:          string(item.High24h),
		LowPrice:           string(item.Low24h),
		Volume:             string(item.BaseVolume),
		BaseVolume:         string(item.BaseVolume),
		QuoteVolume:        string(item.QuoteVolume),
		MarkPrice:          string(item.MarkPrice),
		IndexPrice:         string(item.IndexPrice),
		OpenTime:           timestamp - 24*60*60*1000,
		CloseTime:          timestamp,
	}, nil
}

func (c *Client) FuturesTickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	return c.futuresTickerPrice(ctx, symbol, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresTickerPrice(ctx context.Context, symbol, productType string) (exchange.PriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	query := url.Values{
		"productType": []string{productType},
		"symbol":      []string{normalized},
	}
	var response []contractPriceResponse
	if err := c.getJSON(ctx, "/mix/market/symbol-price", query, &response); err != nil {
		return exchange.PriceTicker{}, err
	}
	if len(response) == 0 {
		return exchange.PriceTicker{}, errors.New("Bitget futures price response is empty")
	}
	timestamp, err := parseOptionalInt(string(response[0].Time))
	if err != nil {
		return exchange.PriceTicker{}, fmt.Errorf("parse futures price timestamp: %w", err)
	}
	return exchange.PriceTicker{Symbol: response[0].Symbol, Price: string(response[0].Price), Time: timestamp}, nil
}

func (c *Client) FuturesBookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	return c.futuresBookTicker(ctx, symbol, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresBookTicker(ctx context.Context, symbol, productType string) (exchange.BookTicker, error) {
	item, err := c.contractTicker(ctx, symbol, productType)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.BookTicker{}, fmt.Errorf("parse futures ticker timestamp: %w", err)
	}
	return exchange.BookTicker{
		Symbol: item.Symbol, BidPrice: string(item.BidPrice), BidQty: string(item.BidQty),
		AskPrice: string(item.AskPrice), AskQty: string(item.AskQty), Time: timestamp,
	}, nil
}

// FuturesPremiumIndex normalizes mark/index/funding data from the contract
// ticker. Bitget exposes all of these values together on its ticker endpoint.
func (c *Client) FuturesPremiumIndex(ctx context.Context, symbol string) (exchange.FuturesPremiumIndex, error) {
	return c.futuresPremiumIndex(ctx, symbol, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresPremiumIndex(ctx context.Context, symbol, productType string) (exchange.FuturesPremiumIndex, error) {
	item, err := c.contractTicker(ctx, symbol, productType)
	if err != nil {
		return exchange.FuturesPremiumIndex{}, err
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.FuturesPremiumIndex{}, fmt.Errorf("parse futures ticker timestamp: %w", err)
	}
	return exchange.FuturesPremiumIndex{
		Symbol:          item.Symbol,
		MarkPrice:       string(item.MarkPrice),
		IndexPrice:      string(item.IndexPrice),
		LastFundingRate: string(item.FundingRate),
		Time:            timestamp,
	}, nil
}

func (c *Client) contractTicker(ctx context.Context, symbol, productType string) (contractTickerResponse, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return contractTickerResponse{}, err
	}
	query := url.Values{
		"productType": []string{productType},
		"symbol":      []string{normalized},
	}
	var response []contractTickerResponse
	if err := c.getJSON(ctx, "/mix/market/ticker", query, &response); err != nil {
		return contractTickerResponse{}, err
	}
	if len(response) == 0 {
		return contractTickerResponse{}, errors.New("Bitget futures ticker response is empty")
	}
	return response[0], nil
}

func decodeLevels(levels [][]json.RawMessage) ([][]string, error) {
	decoded := make([][]string, 0, len(levels))
	for index, level := range levels {
		if len(level) < 2 {
			return nil, fmt.Errorf("level %d has %d fields, want at least 2", index, len(level))
		}
		values := make([]string, 2)
		for field := 0; field < 2; field++ {
			value, err := decodeStringOrNumber(level[field])
			if err != nil {
				return nil, fmt.Errorf("level %d field %d: %w", index, field, err)
			}
			values[field] = value
		}
		decoded = append(decoded, values)
	}
	return decoded, nil
}

func validateContractDepthLimit(limit int) error {
	for _, allowed := range []int{1, 5, 15, 50, 100, 150} {
		if limit == allowed {
			return nil
		}
	}
	return exchange.InvalidParameterError{Parameter: "limit", Message: "must be one of 1, 5, 15, 50, 100 or 150"}
}

func normalizeContractStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "normal":
		return "TRADING"
	case "offline":
		return "OFFLINE"
	case "halt":
		return "HALT"
	default:
		return status
	}
}
