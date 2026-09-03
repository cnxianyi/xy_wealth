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
	"github.com/shopspring/decimal"
)

const maxResponseBytes = 2 << 20

var ErrCredentialsMissing = errors.New("binance API credentials are not configured")

type Client struct {
	baseURL     string
	apiKey      string
	secretKey   string
	recvWindow  int64
	includeZero bool
	httpClient  *http.Client
	now         func() time.Time
}

type accountResponse struct {
	Balances []accountBalance `json:"balances"`
}

type accountBalance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}

type apiError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func New(cfg config.BinanceConfig) *Client {
	return &Client{
		baseURL:     strings.TrimRight(cfg.BaseURL, "/"),
		apiKey:      cfg.APIKey,
		secretKey:   cfg.SecretKey,
		recvWindow:  cfg.RecvWindow,
		includeZero: cfg.IncludeZero,
		httpClient:  &http.Client{Timeout: cfg.HTTPTimeout},
		now:         time.Now,
	}
}

func (c *Client) Name() string { return "binance" }

// Balances calls Binance Spot's signed account endpoint.
func (c *Client) Balances(ctx context.Context) ([]asset.Balance, error) {
	if c.apiKey == "" || c.secretKey == "" {
		return nil, ErrCredentialsMissing
	}

	query := url.Values{}
	query.Set("recvWindow", strconv.FormatInt(c.recvWindow, 10))
	query.Set("timestamp", strconv.FormatInt(c.now().UnixMilli(), 10))
	payload := query.Encode()
	query.Set("signature", sign(payload, c.secretKey))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v3/account?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("create binance account request: %w", err)
	}
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request binance account: %w", unwrapURLError(err))
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read binance account response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var apiErr apiError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Msg != "" {
			return nil, fmt.Errorf("binance API error (status=%d, code=%d): %s", resp.StatusCode, apiErr.Code, apiErr.Msg)
		}
		return nil, fmt.Errorf("binance API error: status=%d", resp.StatusCode)
	}

	var account accountResponse
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, fmt.Errorf("decode binance account response: %w", err)
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
