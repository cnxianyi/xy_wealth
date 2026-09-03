// Package bitget implements Bitget's Spot and futures REST provider.
//
// Bitget uses a response envelope for both public and private endpoints. The
// client unwraps that envelope here so handlers only deal with normalized
// exchange models. Account methods use Classic V2 endpoints when available
// and automatically fall back to the Unified Account (UTA) V3 endpoints when
// Bitget reports error 40085. Futures capabilities are implemented in the
// dedicated contract and coinm files.
package bitget

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
	defaultDepthLimit      = 150
	maxDepthLimit          = 150
	defaultKlinesLimit     = 100
	maxKlinesLimit         = 1000
	bitgetSpotPathPrefix   = "/api/v2"
	bitgetUTAPathPrefix    = "/api/v3"
	bitgetSuccessCode      = "00000"
	accessKeyHeader        = "ACCESS-KEY"
	accessSignHeader       = "ACCESS-SIGN"
	accessTimestampHeader  = "ACCESS-TIMESTAMP"
	accessPassphraseHeader = "ACCESS-PASSPHRASE"
)

var (
	ErrCredentialsMissing = errors.New("bitget API credentials are not configured")
	validIntervals        = map[string]string{
		"1m": "1min", "3m": "3min", "5m": "5min", "15m": "15min", "30m": "30min",
		"1h": "1h", "2h": "2h", "4h": "4h", "6h": "6h", "8h": "8h", "12h": "12h",
		"1d": "1day", "3d": "3day", "1w": "1week", "1M": "1M",
	}
)

// Client is a Bitget REST client. Spot and Mix (futures) market endpoints use
// the configured REST domain; account methods additionally support Bitget's
// Unified Account V3 paths while each capability keeps its own normalization
// logic.
type Client struct {
	baseURL     string
	apiKey      string
	secretKey   string
	passphrase  string
	includeZero bool
	httpClient  *http.Client
	now         func() time.Time
}

// APIError represents an error returned by Bitget or its HTTP gateway.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	if e.Message == "" {
		if e.Code == "" {
			return fmt.Sprintf("bitget API error: status=%d", e.HTTPStatus)
		}
		return fmt.Sprintf("bitget API error (status=%d, code=%s)", e.HTTPStatus, e.Code)
	}
	if e.Code == "" {
		return fmt.Sprintf("bitget API error (status=%d): %s", e.HTTPStatus, e.Message)
	}
	return fmt.Sprintf("bitget API error (status=%d, code=%s): %s", e.HTTPStatus, e.Code, e.Message)
}

type apiResponse[T any] struct {
	Code        json.RawMessage `json:"code"`
	Msg         string          `json:"msg"`
	Message     string          `json:"message"`
	RequestTime int64           `json:"requestTime"`
	Data        T               `json:"data"`
}

type serverTimeResponse struct {
	ServerTime numberString `json:"serverTime"`
}

type symbolResponse struct {
	Symbol              string       `json:"symbol"`
	BaseCoin            string       `json:"baseCoin"`
	QuoteCoin           string       `json:"quoteCoin"`
	MinTradeAmount      numberString `json:"minTradeAmount"`
	MaxTradeAmount      numberString `json:"maxTradeAmount"`
	TakerFeeRate        numberString `json:"takerFeeRate"`
	MakerFeeRate        numberString `json:"makerFeeRate"`
	PricePrecision      intValue     `json:"pricePrecision"`
	QuantityPrecision   intValue     `json:"quantityPrecision"`
	QuotePrecision      intValue     `json:"quotePrecision"`
	Status              string       `json:"status"`
	MinTradeUSDT        numberString `json:"minTradeUSDT"`
	BuyLimitPriceRatio  numberString `json:"buyLimitPriceRatio"`
	SellLimitPriceRatio numberString `json:"sellLimitPriceRatio"`
}

type orderBookResponse struct {
	Asks [][]string   `json:"asks"`
	Bids [][]string   `json:"bids"`
	Time numberString `json:"ts"`
}

type tickerResponse struct {
	Symbol      string       `json:"symbol"`
	High24h     numberString `json:"high24h"`
	Open        numberString `json:"open"`
	Low24h      numberString `json:"low24h"`
	LastPrice   numberString `json:"lastPr"`
	QuoteVolume numberString `json:"quoteVolume"`
	BaseVolume  numberString `json:"baseVolume"`
	BidPrice    numberString `json:"bidPr"`
	AskPrice    numberString `json:"askPr"`
	BidQty      numberString `json:"bidSz"`
	AskQty      numberString `json:"askSz"`
	Time        numberString `json:"ts"`
	Change24h   numberString `json:"change24h"`
}

type accountAssetResponse struct {
	Coin           string       `json:"coin"`
	Available      numberString `json:"available"`
	Frozen         numberString `json:"frozen"`
	Locked         numberString `json:"locked"`
	LimitAvailable numberString `json:"limitAvailable"`
	UpdateTime     numberString `json:"uTime"`
}

type numberString string

func (n *numberString) UnmarshalJSON(raw []byte) error {
	value, err := decodeStringOrNumber(raw)
	if err != nil {
		return err
	}
	*n = numberString(value)
	return nil
}

type intValue int

func (v *intValue) UnmarshalJSON(raw []byte) error {
	value, err := decodeStringOrNumber(raw)
	if err != nil {
		return err
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("expected integer, got %q", value)
	}
	*v = intValue(parsed)
	return nil
}

var _ exchange.SpotProvider = (*Client)(nil)

// New constructs a Bitget provider from configuration.
func New(cfg config.BitgetConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://api.bitget.com"
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		apiKey:      cfg.APIKey,
		secretKey:   cfg.SecretKey,
		passphrase:  cfg.Passphrase,
		includeZero: cfg.IncludeZero,
		httpClient: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		now: time.Now,
	}
}

func (c *Client) Name() string { return "bitget" }

// Ping uses Bitget's public server-time endpoint as the lightweight
// connectivity check; Bitget does not expose a separate Spot ping endpoint.
func (c *Client) Ping(ctx context.Context) error {
	var response serverTimeResponse
	return c.getJSON(ctx, "/public/time", nil, &response)
}

// ServerTime returns Bitget's current server time in Unix milliseconds.
func (c *Client) ServerTime(ctx context.Context) (exchange.ServerTime, error) {
	var response serverTimeResponse
	if err := c.getJSON(ctx, "/public/time", nil, &response); err != nil {
		return exchange.ServerTime{}, err
	}
	serverTime, err := strconv.ParseInt(string(response.ServerTime), 10, 64)
	if err != nil {
		return exchange.ServerTime{}, fmt.Errorf("parse Bitget server time: %w", err)
	}
	return exchange.ServerTime{ServerTime: serverTime}, nil
}

// ExchangeInfo returns Bitget Spot trading-pair rules.
func (c *Client) ExchangeInfo(ctx context.Context, symbol string) (exchange.ExchangeInfo, error) {
	query := url.Values{}
	if strings.TrimSpace(symbol) != "" {
		normalized, err := normalizeSymbol(symbol)
		if err != nil {
			return exchange.ExchangeInfo{}, err
		}
		query.Set("symbol", normalized)
	}
	var response []symbolResponse
	if err := c.getJSON(ctx, "/spot/public/symbols", query, &response); err != nil {
		return exchange.ExchangeInfo{}, err
	}
	info := exchange.ExchangeInfo{
		Timezone: "UTC",
		Symbols:  make([]exchange.SymbolInfo, 0, len(response)),
	}
	for _, item := range response {
		info.Symbols = append(info.Symbols, exchange.SymbolInfo{
			Symbol:              item.Symbol,
			Status:              item.Status,
			BaseAsset:           item.BaseCoin,
			BaseAssetPrecision:  int(item.QuantityPrecision),
			QuoteAsset:          item.QuoteCoin,
			QuotePrecision:      int(item.QuotePrecision),
			QuoteAssetPrecision: int(item.QuotePrecision),
			MinTradeAmount:      string(item.MinTradeAmount),
			MaxTradeAmount:      string(item.MaxTradeAmount),
			MinTradeUSDT:        string(item.MinTradeUSDT),
			TakerFeeRate:        string(item.TakerFeeRate),
			MakerFeeRate:        string(item.MakerFeeRate),
			BuyLimitPriceRatio:  string(item.BuyLimitPriceRatio),
			SellLimitPriceRatio: string(item.SellLimitPriceRatio),
		})
	}
	return info, nil
}

// Depth returns Bitget Spot order-book levels.
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
	query := url.Values{
		"symbol": []string{normalized},
		"type":   []string{"step0"},
		"limit":  []string{strconv.Itoa(limit)},
	}
	var response orderBookResponse
	if err := c.getJSON(ctx, "/spot/market/orderbook", query, &response); err != nil {
		return exchange.OrderBook{}, err
	}
	updateTime, err := parseOptionalInt(string(response.Time))
	if err != nil {
		return exchange.OrderBook{}, fmt.Errorf("parse order book timestamp: %w", err)
	}
	return exchange.OrderBook{Bids: response.Bids, Asks: response.Asks, Time: updateTime}, nil
}

// Klines returns Bitget Spot candlesticks. The shared interface uses Binance
// interval names; this method translates them to Bitget's granularity names.
func (c *Client) Klines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	normalized, err := normalizeSymbol(request.Symbol)
	if err != nil {
		return nil, err
	}
	granularity, ok := validIntervals[strings.TrimSpace(request.Interval)]
	if !ok {
		return nil, exchange.InvalidParameterError{Parameter: "interval", Message: "must be a supported Bitget Spot interval"}
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
	if err := c.getJSON(ctx, "/spot/market/candles", query, &response); err != nil {
		return nil, err
	}
	klines := make([]exchange.Kline, 0, len(response))
	for index, values := range response {
		kline, err := decodeKline(values)
		if err != nil {
			return nil, fmt.Errorf("decode Bitget kline %d: %w", index, err)
		}
		klines = append(klines, kline)
	}
	if len(klines) > limit {
		klines = klines[:limit]
	}
	return klines, nil
}

// Ticker24hr returns Bitget's 24-hour ticker normalized to the shared model.
func (c *Client) Ticker24hr(ctx context.Context, symbol string) (exchange.Ticker24hr, error) {
	item, err := c.ticker(ctx, symbol)
	if err != nil {
		return exchange.Ticker24hr{}, err
	}
	last, err := decimal.NewFromString(string(item.LastPrice))
	if err != nil {
		return exchange.Ticker24hr{}, fmt.Errorf("parse ticker last price: %w", err)
	}
	open, err := decimal.NewFromString(string(item.Open))
	if err != nil {
		return exchange.Ticker24hr{}, fmt.Errorf("parse ticker open price: %w", err)
	}
	priceChange := last.Sub(open)
	priceChangePercent := ""
	if strings.TrimSpace(string(item.Change24h)) != "" {
		change, err := decimal.NewFromString(string(item.Change24h))
		if err != nil {
			return exchange.Ticker24hr{}, fmt.Errorf("parse ticker 24-hour change: %w", err)
		}
		priceChangePercent = change.Mul(decimal.NewFromInt(100)).String()
	} else if !open.IsZero() {
		priceChangePercent = priceChange.Div(open).Mul(decimal.NewFromInt(100)).String()
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.Ticker24hr{}, fmt.Errorf("parse ticker timestamp: %w", err)
	}
	weightedAverage, err := weightedAveragePrice(string(item.BaseVolume), string(item.QuoteVolume))
	if err != nil {
		return exchange.Ticker24hr{}, err
	}
	return exchange.Ticker24hr{
		Symbol:             item.Symbol,
		PriceChange:        priceChange.String(),
		PriceChangePercent: priceChangePercent,
		WeightedAvgPrice:   weightedAverage,
		LastPrice:          string(item.LastPrice),
		BidPrice:           string(item.BidPrice),
		BidQty:             string(item.BidQty),
		AskPrice:           string(item.AskPrice),
		AskQty:             string(item.AskQty),
		OpenPrice:          string(item.Open),
		HighPrice:          string(item.High24h),
		LowPrice:           string(item.Low24h),
		Volume:             string(item.BaseVolume),
		QuoteVolume:        string(item.QuoteVolume),
		CloseTime:          timestamp,
		OpenTime:           timestamp - 24*60*60*1000,
	}, nil
}

// TickerPrice returns Bitget's latest trade price.
func (c *Client) TickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	item, err := c.ticker(ctx, symbol)
	if err != nil {
		return exchange.PriceTicker{}, err
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.PriceTicker{}, fmt.Errorf("parse ticker timestamp: %w", err)
	}
	return exchange.PriceTicker{Symbol: item.Symbol, Price: string(item.LastPrice), Time: timestamp}, nil
}

// BookTicker returns Bitget's best bid and ask from its ticker endpoint.
func (c *Client) BookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	item, err := c.ticker(ctx, symbol)
	if err != nil {
		return exchange.BookTicker{}, err
	}
	timestamp, err := parseOptionalInt(string(item.Time))
	if err != nil {
		return exchange.BookTicker{}, fmt.Errorf("parse ticker timestamp: %w", err)
	}
	return exchange.BookTicker{
		Symbol: item.Symbol, BidPrice: string(item.BidPrice), BidQty: string(item.BidQty),
		AskPrice: string(item.AskPrice), AskQty: string(item.AskQty), Time: timestamp,
	}, nil
}

// Balances calls Bitget Spot's signed account-assets endpoint. It is the
// provider's default account surface used by the existing summary route.
func (c *Client) Balances(ctx context.Context) ([]asset.Balance, error) {
	query := url.Values{"assetType": []string{"all"}}
	var response []accountAssetResponse
	if err := c.getSignedJSON(ctx, "/spot/account/assets", query, &response); err != nil {
		if isUnifiedAccountError(err) {
			return c.utaSpotBalances(ctx)
		}
		return nil, err
	}
	balances := make([]asset.Balance, 0, len(response))
	for _, item := range response {
		available, err := decimal.NewFromString(string(item.Available))
		if err != nil {
			return nil, fmt.Errorf("parse %s available balance: %w", item.Coin, err)
		}
		frozen, err := decimal.NewFromString(string(item.Frozen))
		if err != nil {
			return nil, fmt.Errorf("parse %s frozen balance: %w", item.Coin, err)
		}
		locked, err := decimal.NewFromString(string(item.Locked))
		if err != nil {
			return nil, fmt.Errorf("parse %s locked balance: %w", item.Coin, err)
		}
		limitAvailable, err := decimal.NewFromString(string(item.LimitAvailable))
		if err != nil {
			return nil, fmt.Errorf("parse %s limit available balance: %w", item.Coin, err)
		}
		if !c.includeZero && available.IsZero() && frozen.IsZero() && locked.IsZero() && limitAvailable.IsZero() {
			continue
		}
		balances = append(balances, asset.Balance{
			Symbol: strings.ToUpper(strings.TrimSpace(item.Coin)),
			Free:   available.String(),
			Locked: frozen.Add(locked).String(),
			Total:  available.Add(frozen).Add(locked).String(),
		})
	}
	return balances, nil
}

func (c *Client) ticker(ctx context.Context, symbol string) (tickerResponse, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return tickerResponse{}, err
	}
	var response []tickerResponse
	if err := c.getJSON(ctx, "/spot/market/tickers", url.Values{"symbol": []string{normalized}}, &response); err != nil {
		return tickerResponse{}, err
	}
	if len(response) == 0 {
		return tickerResponse{}, errors.New("Bitget ticker response is empty")
	}
	return response[0], nil
}

func (c *Client) getJSON(ctx context.Context, path string, query url.Values, out any) error {
	return c.doJSON(ctx, http.MethodGet, path, query, false, out)
}

func (c *Client) getSignedJSON(ctx context.Context, path string, query url.Values, out any) error {
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		return ErrCredentialsMissing
	}
	return c.doJSON(ctx, http.MethodGet, path, query, true, out)
}

func (c *Client) getSignedV3JSON(ctx context.Context, path string, query url.Values, out any) error {
	if c.apiKey == "" || c.secretKey == "" || c.passphrase == "" {
		return ErrCredentialsMissing
	}
	return c.doJSONWithPrefix(ctx, http.MethodGet, path, query, true, out, bitgetUTAPathPrefix)
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, signed bool, out any) error {
	return c.doJSONWithPrefix(ctx, method, path, query, signed, out, bitgetSpotPathPrefix)
}

func (c *Client) doJSONWithPrefix(ctx context.Context, method, path string, query url.Values, signed bool, out any, pathPrefix string) error {
	requestURL, err := url.Parse(strings.TrimRight(c.baseURL, "/") + pathPrefix + path)
	if err != nil {
		return fmt.Errorf("create Bitget request URL: %w", err)
	}
	rawQuery := ""
	if query != nil {
		rawQuery = query.Encode()
		requestURL.RawQuery = rawQuery
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), nil)
	if err != nil {
		return fmt.Errorf("create Bitget request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if signed {
		timestamp := strconv.FormatInt(c.now().UnixMilli(), 10)
		queryString := ""
		if rawQuery != "" {
			queryString = "?" + rawQuery
		}
		payload := timestamp + strings.ToUpper(method) + pathPrefix + path + queryString
		req.Header.Set(accessKeyHeader, c.apiKey)
		req.Header.Set(accessSignHeader, sign(payload, c.secretKey))
		req.Header.Set(accessTimestampHeader, timestamp)
		req.Header.Set(accessPassphraseHeader, c.passphrase)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request Bitget: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return fmt.Errorf("read Bitget response: %w", err)
	}
	if len(body) > maxResponseBytes {
		return fmt.Errorf("Bitget response exceeds %d bytes", maxResponseBytes)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return newAPIError(resp.StatusCode, body)
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return errors.New("Bitget response is empty")
	}
	var envelope apiResponse[json.RawMessage]
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decode Bitget response: %w", err)
	}
	code, err := decodeStringOrNumber(envelope.Code)
	if err != nil {
		return fmt.Errorf("decode Bitget response code: %w", err)
	}
	message := envelope.Msg
	if message == "" {
		message = envelope.Message
	}
	if code != bitgetSuccessCode {
		return &APIError{HTTPStatus: resp.StatusCode, Code: code, Message: message}
	}
	if out == nil || len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, out); err != nil {
		return fmt.Errorf("decode Bitget response data: %w", err)
	}
	return nil
}

func newAPIError(status int, body []byte) error {
	var response apiResponse[json.RawMessage]
	_ = json.Unmarshal(body, &response)
	code, _ := decodeStringOrNumber(response.Code)
	message := response.Msg
	if message == "" {
		message = response.Message
	}
	return &APIError{HTTPStatus: status, Code: code, Message: message}
}

func isUnifiedAccountError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.Code == "40085"
}

func sign(payload, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
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

func decodeStringOrNumber(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return "", errors.New("expected string or number")
	}
	if bytes.Equal(raw, []byte("null")) {
		return "", nil
	}
	var stringValue string
	if raw[0] == '"' {
		if err := json.Unmarshal(raw, &stringValue); err != nil {
			return "", err
		}
		return stringValue, nil
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return "", err
	}
	return number.String(), nil
}

func parseOptionalInt(value string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseInt(value, 10, 64)
}

func weightedAveragePrice(baseVolume, quoteVolume string) (string, error) {
	if strings.TrimSpace(baseVolume) == "" || strings.TrimSpace(quoteVolume) == "" {
		return "", nil
	}
	base, err := decimal.NewFromString(baseVolume)
	if err != nil {
		return "", fmt.Errorf("parse base volume: %w", err)
	}
	quote, err := decimal.NewFromString(quoteVolume)
	if err != nil {
		return "", fmt.Errorf("parse quote volume: %w", err)
	}
	if base.IsZero() {
		return "", nil
	}
	return quote.Div(base).String(), nil
}

func decodeKline(values []json.RawMessage) (exchange.Kline, error) {
	if len(values) < 7 {
		return exchange.Kline{}, fmt.Errorf("expected at least 7 fields, got %d", len(values))
	}
	openTime, err := decodeStringOrNumber(values[0])
	if err != nil {
		return exchange.Kline{}, fmt.Errorf("open time: %w", err)
	}
	parsedOpenTime, err := strconv.ParseInt(openTime, 10, 64)
	if err != nil {
		return exchange.Kline{}, fmt.Errorf("open time: %w", err)
	}
	fields := make([]string, 0, len(values)-1)
	for index, value := range values[1:] {
		decoded, err := decodeStringOrNumber(value)
		if err != nil {
			return exchange.Kline{}, fmt.Errorf("field %d: %w", index+1, err)
		}
		fields = append(fields, decoded)
	}
	quoteVolume := fields[5]
	// V2 normally returns both quote-volume and USDT-volume (8 total
	// elements), but older deployments may return only one quote-volume field.
	if len(fields) > 6 && strings.TrimSpace(fields[6]) != "" {
		quoteVolume = fields[6]
	}
	return exchange.Kline{
		OpenTime:                 parsedOpenTime,
		Open:                     fields[0],
		High:                     fields[1],
		Low:                      fields[2],
		Close:                    fields[3],
		Volume:                   fields[4],
		CloseTime:                parsedOpenTime,
		QuoteAssetVolume:         quoteVolume,
		NumberOfTrades:           0,
		TakerBuyBaseAssetVolume:  "0",
		TakerBuyQuoteAssetVolume: "0",
		Ignore:                   "",
	}, nil
}

func unwrapURLError(err error) error {
	var urlError *url.Error
	if errors.As(err, &urlError) && urlError.Err != nil {
		return urlError.Err
	}
	return err
}
