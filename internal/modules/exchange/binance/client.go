package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	maxResponseBytes   = 32 << 20
	maxDepthLimit      = 5000
	maxKlinesLimit     = 1000
	defaultDepthLimit  = 100
	defaultKlinesLimit = 500
)

var (
	ErrCredentialsMissing = errors.New("binance API credentials are not configured")
	validIntervals        = map[string]struct{}{
		"1s": {}, "1m": {}, "3m": {}, "5m": {}, "15m": {}, "30m": {},
		"1h": {}, "2h": {}, "4h": {}, "6h": {}, "8h": {}, "12h": {},
		"1d": {}, "3d": {}, "1w": {}, "1M": {},
	}
)

type Client struct {
	baseURL        string
	futuresBaseURL string
	apiKey         string
	secretKey      string
	recvWindow     int64
	includeZero    bool
	httpClient     *http.Client
	now            func() time.Time
}

type APIError struct {
	HTTPStatus int
	Code       int    `json:"code"`
	Message    string `json:"msg"`
}

func (e *APIError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("binance API error: status=%d", e.HTTPStatus)
	}
	return fmt.Sprintf("binance API error (status=%d, code=%d): %s", e.HTTPStatus, e.Code, e.Message)
}

type accountResponse struct {
	Balances []accountBalance `json:"balances"`
}

type accountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
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
	Symbol                     string           `json:"symbol"`
	Status                     string           `json:"status"`
	BaseAsset                  string           `json:"baseAsset"`
	BaseAssetPrecision         int              `json:"baseAssetPrecision"`
	QuoteAsset                 string           `json:"quoteAsset"`
	QuotePrecision             int              `json:"quotePrecision"`
	QuoteAssetPrecision        int              `json:"quoteAssetPrecision"`
	BaseCommissionPrecision    int              `json:"baseCommissionPrecision"`
	QuoteCommissionPrecision   int              `json:"quoteCommissionPrecision"`
	OrderTypes                 []string         `json:"orderTypes"`
	IcebergAllowed             bool             `json:"icebergAllowed"`
	OCOAllowed                 bool             `json:"ocoAllowed"`
	QuoteOrderQtyMarketAllowed bool             `json:"quoteOrderQtyMarketAllowed"`
	IsSpotTradingAllowed       bool             `json:"isSpotTradingAllowed"`
	IsMarginTradingAllowed     bool             `json:"isMarginTradingAllowed"`
	Filters                    []map[string]any `json:"filters"`
	Permissions                []string         `json:"permissions"`
	PermissionSets             [][]string       `json:"permissionSets"`
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
	WeightedAvgPrice   string `json:"weightedAvgPrice"`
	PrevClosePrice     string `json:"prevClosePrice"`
	LastPrice          string `json:"lastPrice"`
	LastQty            string `json:"lastQty"`
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
	FirstID            int64  `json:"firstId"`
	LastID             int64  `json:"lastId"`
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

func New(cfg config.BinanceConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	recvWindow := cfg.RecvWindow
	if recvWindow <= 0 {
		recvWindow = 5000
	}
	futuresBaseURL := cfg.FuturesBaseURL
	if futuresBaseURL == "" {
		futuresBaseURL = "https://fapi.binance.com"
	}
	return &Client{
		baseURL:        strings.TrimRight(cfg.BaseURL, "/"),
		futuresBaseURL: strings.TrimRight(futuresBaseURL, "/"),
		apiKey:         cfg.APIKey,
		secretKey:      cfg.SecretKey,
		recvWindow:     recvWindow,
		includeZero:    cfg.IncludeZero,
		httpClient:     &http.Client{Timeout: timeout},
		now:            time.Now,
	}
}

func (c *Client) Name() string { return "binance" }

// Ping tests connectivity to Binance's Spot REST API.
func (c *Client) Ping(ctx context.Context) error {
	return c.getJSON(ctx, "/api/v3/ping", nil, nil)
}

// ServerTime returns Binance's current server time in milliseconds.
func (c *Client) ServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getJSON(ctx, "/api/v3/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	return exchange.ServerTime{ServerTime: response.ServerTime}, nil
}

// ExchangeInfo returns Spot trading rules. An empty symbol requests all
// symbols; a symbol limits the response to that trading pair.
func (c *Client) ExchangeInfo(ctx context.Context, symbol string) (exchange.ExchangeInfo, error) {
	query := url.Values{}
	if symbol != "" {
		normalized, err := normalizeSymbol(symbol)
		if err != nil {
			return exchange.ExchangeInfo{}, err
		}
		query.Set("symbol", normalized)
	}

	var response exchangeInfoResponse
	if err := c.getJSON(ctx, "/api/v3/exchangeInfo", query, &response); err != nil {
		return exchange.ExchangeInfo{}, err
	}

	info := exchange.ExchangeInfo{
		Timezone:        response.Timezone,
		ServerTime:      response.ServerTime,
		ExchangeFilters: response.ExchangeFilters,
		Symbols:         make([]exchange.SymbolInfo, 0, len(response.Symbols)),
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
	for _, symbolInfo := range response.Symbols {
		info.Symbols = append(info.Symbols, exchange.SymbolInfo{
			Symbol:                     symbolInfo.Symbol,
			Status:                     symbolInfo.Status,
			BaseAsset:                  symbolInfo.BaseAsset,
			BaseAssetPrecision:         symbolInfo.BaseAssetPrecision,
			QuoteAsset:                 symbolInfo.QuoteAsset,
			QuotePrecision:             symbolInfo.QuotePrecision,
			QuoteAssetPrecision:        symbolInfo.QuoteAssetPrecision,
			BaseCommissionPrecision:    symbolInfo.BaseCommissionPrecision,
			QuoteCommissionPrecision:   symbolInfo.QuoteCommissionPrecision,
			OrderTypes:                 symbolInfo.OrderTypes,
			IcebergAllowed:             symbolInfo.IcebergAllowed,
			OCOAllowed:                 symbolInfo.OCOAllowed,
			QuoteOrderQtyMarketAllowed: symbolInfo.QuoteOrderQtyMarketAllowed,
			IsSpotTradingAllowed:       symbolInfo.IsSpotTradingAllowed,
			IsMarginTradingAllowed:     symbolInfo.IsMarginTradingAllowed,
			Filters:                    symbolInfo.Filters,
			Permissions:                symbolInfo.Permissions,
			PermissionSets:             symbolInfo.PermissionSets,
		})
	}
	return info, nil
}

// Depth returns the current order book for a Spot symbol.
func (c *Client) Depth(ctx context.Context, symbol string, limit int) (exchange.OrderBook, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.OrderBook{}, err
	}
	if limit == 0 {
		limit = defaultDepthLimit
	}
	if err := validateLimit("limit", limit, maxDepthLimit); err != nil {
		return exchange.OrderBook{}, err
	}

	query := url.Values{"symbol": []string{normalized}, "limit": []string{strconv.Itoa(limit)}}
	var response orderBookResponse
	if err := c.getJSON(ctx, "/api/v3/depth", query, &response); err != nil {
		return exchange.OrderBook{}, err
	}
	return exchange.OrderBook{
		LastUpdateID: response.LastUpdateID,
		Bids:         response.Bids,
		Asks:         response.Asks,
	}, nil
}

// Klines returns candlesticks for a Spot symbol and interval.
func (c *Client) Klines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	interval := strings.TrimSpace(request.Interval)
	if _, ok := validIntervals[interval]; !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported Binance interval"}
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
	if timezone := strings.TrimSpace(request.TimeZone); timezone != "" {
		query.Set("timeZone", timezone)
	}

	var raw []json.RawMessage
	if err := c.getJSON(ctx, "/api/v3/klines", query, &raw); err != nil {
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
	return klines, nil
}

// Ticker24hr returns the rolling 24-hour price change statistics.
func (c *Client) Ticker24hr(ctx context.Context, symbol string) (exchange.Ticker24hr, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.Ticker24hr{}, err
	}
	var response ticker24hrResponse
	if err := c.getJSON(ctx, "/api/v3/ticker/24hr", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.Ticker24hr{}, err
	}
	return exchange.Ticker24hr{
		Symbol:             response.Symbol,
		PriceChange:        response.PriceChange,
		PriceChangePercent: response.PriceChangePercent,
		WeightedAvgPrice:   response.WeightedAvgPrice,
		PrevClosePrice:     response.PrevClosePrice,
		LastPrice:          response.LastPrice,
		LastQty:            response.LastQty,
		BidPrice:           response.BidPrice,
		BidQty:             response.BidQty,
		AskPrice:           response.AskPrice,
		AskQty:             response.AskQty,
		OpenPrice:          response.OpenPrice,
		HighPrice:          response.HighPrice,
		LowPrice:           response.LowPrice,
		Volume:             response.Volume,
		QuoteVolume:        response.QuoteVolume,
		OpenTime:           response.OpenTime,
		CloseTime:          response.CloseTime,
		FirstID:            response.FirstID,
		LastID:             response.LastID,
		Count:              response.Count,
	}, nil
}

// TickerPrice returns the latest price for a Spot symbol.
func (c *Client) TickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	var response priceTickerResponse
	if err := c.getJSON(ctx, "/api/v3/ticker/price", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return exchange.PriceTicker{}, err
	}
	return exchange.PriceTicker{Symbol: response.Symbol, Price: response.Price}, nil
}

// BookTicker returns the best bid and ask for a Spot symbol.
func (c *Client) BookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	var response bookTickerResponse
	if err := c.getJSON(ctx, "/api/v3/ticker/bookTicker", url.Values{"symbol": []string{normalized}}, &response); err != nil {
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

// Balances calls Binance Spot's signed account endpoint.
func (c *Client) Balances(ctx context.Context) ([]asset.Balance, error) {
	query := url.Values{}
	var account accountResponse
	if err := c.getSignedJSON(ctx, "/api/v3/account", query, &account); err != nil {
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
	if c.apiKey == "" || c.secretKey == "" {
		return ErrCredentialsMissing
	}
	if query == nil {
		query = url.Values{}
	}
	query.Set("recvWindow", strconv.FormatInt(c.recvWindow, 10))
	query.Set("timestamp", strconv.FormatInt(c.now().UnixMilli(), 10))
	payload := query.Encode()
	query.Set("signature", sign(payload, c.secretKey))

	return c.doJSON(ctx, c.baseURL, http.MethodGet, path, query, true, out)
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	return c.doJSON(ctx, c.baseURL, http.MethodGet, path, query, false, out)
}

func (c *Client) getFuturesJSON(ctx context.Context, path string, query url.Values, out any) error {
	return c.doJSON(ctx, c.futuresBaseURL, http.MethodGet, path, query, false, out)
}

func (c *Client) doJSON(ctx context.Context, baseURL, method, path string, query url.Values, signed bool, out any) error {
	requestURL, err := url.Parse(baseURL + path)
	if err != nil {
		return fmt.Errorf("create binance request URL: %w", err)
	}
	requestURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create binance request: %w", err)
	}
	if signed {
		req.Header.Set("X-MBX-APIKEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request binance: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read binance response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("binance response exceeds %d bytes", maxResponseBytes)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var responseError struct {
			Code int    `json:"code"`
			Msg  string `json:"msg"`
		}
		_ = json.Unmarshal(body, &responseError)
		return &APIError{HTTPStatus: resp.StatusCode, Code: responseError.Code, Message: responseError.Msg}
	}
	if out == nil || len(body) == 0 {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("decode binance response: %w", err)
	}
	return nil
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
	if len(values) < 12 {
		return exchange.Kline{}, fmt.Errorf("expected 12 fields, got %d", len(values))
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
	fields := make([]string, 0, 9)
	for _, index := range []int{1, 2, 3, 4, 5, 7, 9, 10, 11} {
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
		Ignore:                   fields[8],
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

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
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
