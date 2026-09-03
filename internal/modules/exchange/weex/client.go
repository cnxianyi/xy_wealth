// Package weex implements the Weex V3 Spot REST provider.
//
// Spot and Contract use different upstream domains. The Spot capability is
// implemented here first; Contract methods will be added without coupling the
// two API surfaces together.
package weex

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

const (
	maxResponseBytes       = 32 << 20
	defaultDepthLimit      = 15
	defaultKlinesLimit     = 100
	maxKlinesLimit         = 1000
	weexSpotPathPrefix     = "/api/v3"
	accessKeyHeader        = "ACCESS-KEY"
	accessSignHeader       = "ACCESS-SIGN"
	accessTimestampHeader  = "ACCESS-TIMESTAMP"
	accessPassphraseHeader = "ACCESS-PASSPHRASE"
)

var (
	ErrCredentialsMissing = errors.New("weex API credentials are not configured")
	validIntervals        = map[string]struct{}{
		"1m": {}, "5m": {}, "15m": {}, "30m": {},
		"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
		"1d": {}, "1w": {},
	}
	validDepthLimits = map[int]struct{}{15: {}, 200: {}}
)

// Client is a Weex API client. Its Spot and Contract base URLs stay separate
// so that adding Contract support cannot accidentally send requests to Spot.
type Client struct {
	spotBaseURL     string
	contractBaseURL string
	apiKey          string
	secretKey       string
	passphrase      string
	includeZero     bool
	httpClient      *http.Client
	now             func() time.Time
}

// APIError represents an error response returned by Weex.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		if e.Code == "" {
			return fmt.Sprintf("weex API error: status=%d", e.HTTPStatus)
		}
		return fmt.Sprintf("weex API error (status=%d, code=%s)", e.HTTPStatus, e.Code)
	}
	if e.Code == "" {
		return fmt.Sprintf("weex API error (status=%d): %s", e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("weex API error (status=%d, code=%s): %s", e.HTTPStatus, e.Code, e.Message)
}

type serverTimeResponse struct {
	ServerTime int64 `json:"serverTime"`
}

type exchangeInfoResponse struct {
	Timezone        string               `json:"timezone"`
	ServerTime      int64                `json:"serverTime"`
	RateLimits      []rateLimitResponse  `json:"rateLimits"`
	ExchangeFilters []map[string]any     `json:"exchangeFilters"`
	Symbols         []symbolInfoResponse `json:"symbols"`
}

type rateLimitResponse struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
}

type symbolInfoResponse struct {
	Symbol                   string       `json:"symbol"`
	Status                   string       `json:"status"`
	BaseAsset                string       `json:"baseAsset"`
	BaseAssetPrecision       int          `json:"baseAssetPrecision"`
	QuoteAsset               string       `json:"quoteAsset"`
	QuoteAssetPrecision      int          `json:"quoteAssetPrecision"`
	TickSize                 numberString `json:"tickSize"`
	StepSize                 numberString `json:"stepSize"`
	MinTradeAmount           numberString `json:"minTradeAmount"`
	MaxTradeAmount           numberString `json:"maxTradeAmount"`
	TakerFeeRate             numberString `json:"takerFeeRate"`
	MakerFeeRate             numberString `json:"makerFeeRate"`
	BuyLimitPriceRatio       numberString `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio      numberString `json:"sellLimitPriceRatio"`
	MarketBuyLimitSize       numberString `json:"marketBuyLimitSize"`
	MarketSellLimitSize      numberString `json:"marketSellLimitSize"`
	MarketFallbackPriceRatio numberString `json:"marketFallbackPriceRatio"`
	EnableTrade              *bool        `json:"enableTrade"`
	EnableDisplay            *bool        `json:"enableDisplay"`
	DisplayDigitMerge        string       `json:"displayDigitMerge"`
	DisplayNew               *bool        `json:"displayNew"`
	DisplayHot               *bool        `json:"displayHot"`
}

type orderBookResponse struct {
	LastUpdateID int64      `json:"lastUpdateId"`
	Bids         [][]string `json:"bids"`
	Asks         [][]string `json:"asks"`
}

type ticker24hrResponse struct {
	Symbol             string `json:"symbol"`
	PriceChange        string `json:"priceChange"`
	PriceChangePercent string `json:"priceChangePercent"`
	LastPrice          string `json:"lastPrice"`
	BidPrice           string `json:"bidPrice"`
	BidQty             string `json:"bidQty"`
	AskPrice           string `json:"askPrice"`
	AskQty             string `json:"askQty"`
	OpenPrice          string `json:"openPrice"`
	HighPrice          string `json:"highPrice"`
	LowPrice           string `json:"lowPrice"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"`
	OpenTime           int64  `json:"openTime"`
	CloseTime          int64  `json:"closeTime"`
	Count              int64  `json:"count"`
}

type priceTickerResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type bookTickerResponse struct {
	Symbol   string `json:"symbol"`
	BidPrice string `json:"bidPrice"`
	BidQty   string `json:"bidQty"`
	AskPrice string `json:"askPrice"`
	AskQty   string `json:"askQty"`
}

type accountResponse struct {
	Balances []accountBalance `json:"balances"`
}

type accountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

// numberString preserves a decimal returned either as a JSON string or as a
// JSON number. Keeping it as text avoids introducing floating-point rounding.
type numberString string

func (n *numberString) UnmarshalJSON(raw []byte) error {
	value, err := decodeJSONNumberString(raw)
	if err != nil {
		return err
	}
	*n = numberString(value)
	return nil
}

// responseList handles endpoints that return an object for a symbol query and
// an array when the upstream decides to return multiple symbols.
type responseList[T any] []T

func (r *responseList[T]) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		*r = nil
		return nil
	}
	if raw[0] == '[' {
		var values []T
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		*r = values
		return nil
	}
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	*r = []T{value}
	return nil
}

var _ exchange.SpotProvider = (*Client)(nil)

// New constructs a Weex provider from configuration.
func New(cfg config.WeexConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	spotBaseURL := cfg.SpotBaseURL
	if spotBaseURL == "" {
		spotBaseURL = "https://api-spot.weex.com"
	}
	contractBaseURL := cfg.ContractBaseURL
	if contractBaseURL == "" {
		contractBaseURL = "https://api-contract.weex.com"
	}
	return &Client{
		spotBaseURL:     strings.TrimRight(spotBaseURL, "/"),
		contractBaseURL: strings.TrimRight(contractBaseURL, "/"),
		apiKey:          cfg.APIKey,
		secretKey:       cfg.SecretKey,
		passphrase:      cfg.Passphrase,
		includeZero:     cfg.IncludeZero,
		httpClient: &http.Client{
			Timeout: timeout,
			// Do not forward signed Weex headers to a redirected host.
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (c *Client) Name() string { return "weex" }

// Ping tests connectivity to Weex Spot's REST API.
func (c *Client) Ping(ctx context.Context) error {
	return c.getJSON(ctx, weexSpotPathPrefix+"/ping", nil, nil)
}

// ServerTime returns Weex Spot's current server time in milliseconds.
func (c *Client) ServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	return exchange.ServerTime{ServerTime: response.ServerTime}, nil
}

// ExchangeInfo returns Weex Spot trading rules. An empty symbol requests all
// symbols; a symbol limits the response when supported by the upstream.
func (c *Client) ExchangeInfo(ctx context.Context, symbol string) (exchange.ExchangeInfo, error) {
	query := url.Values{}
	if strings.TrimSpace(symbol) != "" {
		normalized, err := normalizeSymbol(symbol)
		if err != nil {
			return exchange.ExchangeInfo{}, err
		}
		query.Set("symbol", normalized)
	}

	var response exchangeInfoResponse
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/exchangeInfo", query, &response); err != nil {
		return exchange.ExchangeInfo{}, err
	}

	info := exchange.ExchangeInfo{
		Timezone:        response.Timezone,
		ServerTime:      response.ServerTime,
		ExchangeFilters: response.ExchangeFilters,
		RateLimits:      make([]exchange.RateLimit, 0, len(response.RateLimits)),
		Symbols:         make([]exchange.SymbolInfo, 0, len(response.Symbols)),
	}
	for _, limit := range response.RateLimits {
		info.RateLimits = append(info.RateLimits, exchange.RateLimit{
			RateLimitType: limit.RateLimitType,
			Interval:      limit.Interval,
			IntervalNum:   limit.IntervalNum,
			Limit:         limit.Limit,
		})
	}
	for _, symbolInfo := range response.Symbols {
		info.Symbols = append(info.Symbols, exchange.SymbolInfo{
			Symbol:                   symbolInfo.Symbol,
			Status:                   symbolInfo.Status,
			BaseAsset:                symbolInfo.BaseAsset,
			BaseAssetPrecision:       symbolInfo.BaseAssetPrecision,
			QuoteAsset:               symbolInfo.QuoteAsset,
			QuoteAssetPrecision:      symbolInfo.QuoteAssetPrecision,
			TickSize:                 string(symbolInfo.TickSize),
			StepSize:                 string(symbolInfo.StepSize),
			MinTradeAmount:           string(symbolInfo.MinTradeAmount),
			MaxTradeAmount:           string(symbolInfo.MaxTradeAmount),
			TakerFeeRate:             string(symbolInfo.TakerFeeRate),
			MakerFeeRate:             string(symbolInfo.MakerFeeRate),
			BuyLimitPriceRatio:       string(symbolInfo.BuyLimitPriceRatio),
			SellLimitPriceRatio:      string(symbolInfo.SellLimitPriceRatio),
			MarketBuyLimitSize:       string(symbolInfo.MarketBuyLimitSize),
			MarketSellLimitSize:      string(symbolInfo.MarketSellLimitSize),
			MarketFallbackPriceRatio: string(symbolInfo.MarketFallbackPriceRatio),
			EnableTrade:              symbolInfo.EnableTrade,
			EnableDisplay:            symbolInfo.EnableDisplay,
			DisplayDigitMerge:        symbolInfo.DisplayDigitMerge,
			DisplayNew:               symbolInfo.DisplayNew,
			DisplayHot:               symbolInfo.DisplayHot,
		})
	}
	return info, nil
}

// Depth returns the current Weex Spot order book.
func (c *Client) Depth(ctx context.Context, symbol string, limit int) (exchange.OrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.OrderBook{}, err
	}
	if limit == 0 {
		limit = defaultDepthLimit
	}
	if _, ok := validDepthLimits[limit]; !ok {
		return exchange.OrderBook{}, exchange.InvalidParameterError{
			Parameter: "limit",
			Message:   "must be one of 15 or 200",
		}
	}

	query := url.Values{"symbol": []string{normalized}, "limit": []string{strconv.Itoa(limit)}}
	var response orderBookResponse
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/market/depth", query, &response); err != nil {
		return exchange.OrderBook{}, err
	}
	return exchange.OrderBook{
		LastUpdateID: response.LastUpdateID,
		Bids:         response.Bids,
		Asks:         response.Asks,
	}, nil
}

// Klines returns candlesticks for a Weex Spot symbol and interval. Weex does
// not define Binance's timeZone parameter, so it is intentionally omitted.
func (c *Client) Klines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(request.Interval)
	if _, ok := validIntervals[interval]; !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported Weex Spot interval"}
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
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/market/klines", query, &raw); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(raw))
	for index, item := range raw {
		kline, err := decodeKline(item)
		if err != nil {
			return nil, fmt.Errorf("decode kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	// Some Weex deployments return their full recent window even when limit is
	// supplied. Enforce the provider contract locally so callers get the
	// requested number of candles and response sizes stay bounded.
	if len(klines) > limit {
		klines = klines[:limit]
	}
	return klines, nil
}

// Ticker24hr returns the rolling 24-hour price change statistics.
func (c *Client) Ticker24hr(ctx context.Context, symbol string) (exchange.Ticker24hr, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.Ticker24hr{}, err
	}
	var response responseList[ticker24hrResponse]
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/market/ticker/24hr", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.Ticker24hr{}, err
	}
	if len(response) == 0 {
		return exchange.Ticker24hr{}, errors.New("weex 24-hour ticker response is empty")
	}
	item := response[0]
	return exchange.Ticker24hr{
		Symbol:             item.Symbol,
		PriceChange:        item.PriceChange,
		PriceChangePercent: item.PriceChangePercent,
		WeightedAvgPrice:   calculateWeightedAvgPrice(item.Volume, item.QuoteVolume),
		LastPrice:          item.LastPrice,
		BidPrice:           item.BidPrice,
		BidQty:             item.BidQty,
		AskPrice:           item.AskPrice,
		AskQty:             item.AskQty,
		OpenPrice:          item.OpenPrice,
		HighPrice:          item.HighPrice,
		LowPrice:           item.LowPrice,
		Volume:             item.Volume,
		QuoteVolume:        item.QuoteVolume,
		OpenTime:           item.OpenTime,
		CloseTime:          item.CloseTime,
		Count:              item.Count,
	}, nil
}

// TickerPrice returns the latest price for a Weex Spot symbol.
func (c *Client) TickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	var response responseList[priceTickerResponse]
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/market/ticker/price", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.PriceTicker{}, err
	}
	if len(response) == 0 {
		return exchange.PriceTicker{}, errors.New("weex ticker price response is empty")
	}
	return exchange.PriceTicker{Symbol: response[0].Symbol, Price: response[0].Price}, nil
}

// BookTicker returns the best bid and ask for a Weex Spot symbol.
func (c *Client) BookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	var response responseList[bookTickerResponse]
	if err := c.getJSON(ctx, weexSpotPathPrefix+"/market/ticker/bookTicker", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.BookTicker{}, err
	}
	if len(response) == 0 {
		return exchange.BookTicker{}, errors.New("weex book ticker response is empty")
	}
	item := response[0]
	return exchange.BookTicker{
		Symbol:   item.Symbol,
		BidPrice: item.BidPrice,
		BidQty:   item.BidQty,
		AskPrice: item.AskPrice,
		AskQty:   item.AskQty,
	}, nil
}

// Balances calls Weex Spot's signed account endpoint.
func (c *Client) Balances(ctx context.Context) ([]asset.Balance, error) {
	var account accountResponse
	if err := c.getSignedJSON(ctx, weexSpotPathPrefix+"/account", nil, &account); err != nil {
		return nil, err
	}

	balances := make([]asset.Balance, 0, len(account.Balances))
	for _, item := range account.Balances {
		free, err := decimal.NewFromString(item.Free)
		if err != nil {
			return nil, fmt.Errorf("parse %s free balance: %w", item.Asset, err)
		}
		locked, err := decimal.NewFromString(item.Locked)
		if err != nil {
			return nil, fmt.Errorf("parse %s locked balance: %w", item.Asset, err)
		}
		if !c.includeZero && free.IsZero() && locked.IsZero() {
			continue
		}
		balances = append(balances, asset.Balance{
			Symbol: item.Asset,
			Free:   free.String(),
			Locked: locked.String(),
			Total:  free.Add(locked).String(),
		})
	}
	return balances, nil
}

func (c *Client) getSignedJSON(ctx context.Context, path string, query url.Values, out any) error {
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		return ErrCredentialsMissing
	}
	return c.doJSON(ctx, c.spotBaseURL, http.MethodGet, path, query, true, out)
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	return c.doJSON(ctx, c.spotBaseURL, http.MethodGet, path, query, false, out)
}

func (c *Client) doJSON(ctx context.Context, baseURL, method, path string, query url.Values, signed bool, out any) error {
	requestURL, err := url.Parse(strings.TrimRight(baseURL, "/") + path)
	if err != nil {
		return fmt.Errorf("create weex request URL: %w", err)
	}
	rawQuery := ""
	if query != nil {
		rawQuery = query.Encode()
	}
	requestURL.RawQuery = rawQuery
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create weex request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if signed {
		timestamp := strconv.FormatInt(c.now().UnixMilli(), 10)
		queryString := ""
		if rawQuery != "" {
			queryString = "?" + rawQuery
		}
		payload := timestamp + strings.ToUpper(method) + path + queryString
		req.Header.Set(accessKeyHeader, c.apiKey)
		req.Header.Set(accessSignHeader, sign(payload, c.secretKey))
		req.Header.Set(accessTimestampHeader, timestamp)
		req.Header.Set(accessPassphraseHeader, c.passphrase)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request weex: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read weex response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("weex response exceeds %d bytes", maxResponseBytes)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(resp.StatusCode, body)
	}
	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode weex response: %w", err)
	}
	return nil
}

func newAPIError(status int, body []byte) error {
	var response struct {
		Code    json.RawMessage `json:"code"`
		Msg     string          `json:"msg"`
		Message string          `json:"message"`
	}
	_ = json.Unmarshal(body, &response)
	code, _ := decodeJSONNumberString(response.Code)
	message := response.Msg
	if message == "" {
		message = response.Message
	}
	return &APIError{HTTPStatus: status, Code: code, Message: message}
}

func normalizeSymbol(symbol string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(symbol))
	if normalized == "" {
		return "", exchange.InvalidParameterError{Parameter: "symbol", Message: "is required"}
	}
	return normalized, nil
}

func validateLimit(parameter string, limit, maximum int) error {
	if limit < 1 || limit > maximum {
		return exchange.InvalidParameterError{Parameter: parameter, Message: fmt.Sprintf("must be between 1 and %d", maximum)}
	}
	return nil
}

func decodeKline(raw json.RawMessage) (exchange.Kline, error) {
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil {
		return exchange.Kline{}, err
	}
	if len(values) < 11 {
		return exchange.Kline{}, fmt.Errorf("expected 11 fields, got %d", len(values))
	}

	openTime, err := decodeInt(values[0])
	if err != nil {
		return exchange.Kline{}, fmt.Errorf("open time: %w", err)
	}
	closeTime, err := decodeInt(values[6])
	if err != nil {
		return exchange.Kline{}, fmt.Errorf("close time: %w", err)
	}
	numberOfTrades, err := decodeInt(values[8])
	if err != nil {
		return exchange.Kline{}, fmt.Errorf("number of trades: %w", err)
	}
	fields := make([]string, 0, 8)
	for _, index := range []int{1, 2, 3, 4, 5, 7, 9, 10} {
		value, err := decodeString(values[index])
		if err != nil {
			return exchange.Kline{}, fmt.Errorf("field %d: %w", index, err)
		}
		fields = append(fields, value)
	}
	return exchange.Kline{
		OpenTime:                 openTime,
		Open:                     fields[0],
		High:                     fields[1],
		Low:                      fields[2],
		Close:                    fields[3],
		Volume:                   fields[4],
		CloseTime:                closeTime,
		QuoteAssetVolume:         fields[5],
		NumberOfTrades:           numberOfTrades,
		TakerBuyBaseAssetVolume:  fields[6],
		TakerBuyQuoteAssetVolume: fields[7],
		Ignore:                   "",
	}, nil
}

func decodeInt(raw json.RawMessage) (int64, error) {
	var number int64
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, nil
	}
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		return strconv.ParseInt(stringValue, 10, 64)
	}
	return 0, fmt.Errorf("expected integer")
}

func decodeString(raw json.RawMessage) (string, error) {
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		return stringValue, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err == nil {
		return number.String(), nil
	}
	return "", fmt.Errorf("expected string or number")
}

func decodeJSONNumberString(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return "", nil
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return "", err
		}
		return value, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", err
	}
	return number.String(), nil
}

func calculateWeightedAvgPrice(volume, quoteVolume string) string {
	volumeDecimal, err := decimal.NewFromString(volume)
	if err != nil || volumeDecimal.IsZero() {
		return ""
	}
	quoteDecimal, err := decimal.NewFromString(quoteVolume)
	if err != nil {
		return ""
	}
	return quoteDecimal.Div(volumeDecimal).String()
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func unwrapURLError(err error) error {
	for {
		var urlErr *url.Error
		if !errors.As(err, &urlErr) {
			return err
		}
		if urlErr.Err == nil {
			return errors.New("network request failed")
		}
		err = urlErr.Err
	}
}
