package binance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

type futuresAccountBalanceResponse struct {
	AccountAlias          string `json:"accountAlias"`
	Asset                 string `json:"asset"`
	Balance               string `json:"balance"`
	WithdrawAvailable     string `json:"withdrawAvailable"`
	CrossWalletBalance    string `json:"crossWalletBalance"`
	CrossUnrealizedProfit string `json:"crossUnPnl"`
	AvailableBalance      string `json:"availableBalance"`
	MaxWithdrawAmount     string `json:"maxWithdrawAmount"`
	MarginAvailable       *bool  `json:"marginAvailable"`
	UpdateTime            int64  `json:"updateTime"`
}

type futuresPositionResponse struct {
	Symbol                 string     `json:"symbol"`
	Pair                   string     `json:"pair"`
	PositionSide           string     `json:"positionSide"`
	PositionAmount         string     `json:"positionAmt"`
	EntryPrice             string     `json:"entryPrice"`
	BreakEvenPrice         string     `json:"breakEvenPrice"`
	MarkPrice              string     `json:"markPrice"`
	UnrealizedProfit       string     `json:"unRealizedProfit"`
	LiquidationPrice       string     `json:"liquidationPrice"`
	Leverage               string     `json:"leverage"`
	MarginType             string     `json:"marginType"`
	IsolatedMargin         string     `json:"isolatedMargin"`
	IsAutoAddMargin        *boolValue `json:"isAutoAddMargin"`
	Notional               string     `json:"notional"`
	NotionalValue          string     `json:"notionalValue"`
	MarginAsset            string     `json:"marginAsset"`
	IsolatedWallet         string     `json:"isolatedWallet"`
	InitialMargin          string     `json:"initialMargin"`
	MaintenanceMargin      string     `json:"maintMargin"`
	PositionInitialMargin  string     `json:"positionInitialMargin"`
	OpenOrderInitialMargin string     `json:"openOrderInitialMargin"`
	MaxNotionalValue       string     `json:"maxNotionalValue"`
	MaxQuantity            string     `json:"maxQty"`
	BidNotional            string     `json:"bidNotional"`
	AskNotional            string     `json:"askNotional"`
	ADL                    int64      `json:"adl"`
	UpdateTime             int64      `json:"updateTime"`
}

// boolValue accepts both the boolean and string representations returned by
// Binance's futures position endpoints.
type boolValue bool

func (v *boolValue) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	var boolean bool
	if err := json.Unmarshal(raw, &boolean); err == nil {
		*v = boolValue(boolean)
		return nil
	}
	var stringValue string
	if err := json.Unmarshal(raw, &stringValue); err == nil {
		parsed, err := strconv.ParseBool(stringValue)
		if err != nil {
			return fmt.Errorf("expected boolean, got %q", stringValue)
		}
		*v = boolValue(parsed)
		return nil
	}
	return fmt.Errorf("expected boolean or boolean string")
}

var _ exchange.USDSMFuturesAccountProvider = (*Client)(nil)
var _ exchange.COINMFuturesAccountProvider = (*Client)(nil)

// USDSMFuturesAccountBalances returns Binance USDⓈ-M Futures account assets.
func (c *Client) USDSMFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	var response []futuresAccountBalanceResponse
	if err := c.getSignedFuturesJSON(ctx, "/fapi/v3/balance", nil, &response); err != nil {
		return nil, err
	}
	return normalizeFuturesAccountBalances(c.includeZero, response)
}

// COINMFuturesAccountBalances returns Binance COIN-M Futures account assets.
func (c *Client) COINMFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	var response []futuresAccountBalanceResponse
	if err := c.getSignedCoinMFuturesJSON(ctx, "/dapi/v1/balance", nil, &response); err != nil {
		return nil, err
	}
	return normalizeFuturesAccountBalances(c.includeZero, response)
}

// USDSMFuturesPositions returns Binance USDⓈ-M Futures positions. An empty
// symbol asks Binance for all symbols with a position or open order.
func (c *Client) USDSMFuturesPositions(ctx context.Context, symbol string) ([]exchange.FuturesPosition, error) {
	query, err := futuresPositionQuery(symbol)
	if err != nil {
		return nil, err
	}
	var response []futuresPositionResponse
	if err := c.getSignedFuturesJSON(ctx, "/fapi/v3/positionRisk", query, &response); err != nil {
		return nil, err
	}
	return normalizeFuturesPositions(c.includeZero, response), nil
}

// COINMFuturesPositions returns Binance COIN-M Futures positions. An empty
// symbol asks Binance for all symbols; a contract symbol is normalized and
// converted to the pair filter required by Binance (for example,
// BTCUSD_PERP becomes pair=BTCUSD).
func (c *Client) COINMFuturesPositions(ctx context.Context, symbol string) ([]exchange.FuturesPosition, error) {
	query, err := coinMFuturesPositionQuery(symbol)
	if err != nil {
		return nil, err
	}
	var response []futuresPositionResponse
	if err := c.getSignedCoinMFuturesJSON(ctx, "/dapi/v1/positionRisk", query, &response); err != nil {
		return nil, err
	}
	return normalizeFuturesPositions(c.includeZero, response), nil
}

func futuresPositionQuery(symbol string) (url.Values, error) {
	query := url.Values{}
	if strings.TrimSpace(symbol) == "" {
		return query, nil
	}
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return nil, err
	}
	query.Set("symbol", normalized)
	return query, nil
}

func coinMFuturesPositionQuery(symbol string) (url.Values, error) {
	query := url.Values{}
	if strings.TrimSpace(symbol) == "" {
		return query, nil
	}
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return nil, err
	}
	// Binance's COIN-M positionRisk endpoint filters by pair (for example,
	// BTCUSD), while the public API accepts a contract symbol such as
	// BTCUSD_PERP or BTCUSD_240628. Strip the contract suffix before signing
	// the upstream request.
	pair := normalized
	if separator := strings.IndexByte(pair, '_'); separator > 0 {
		pair = pair[:separator]
	}
	query.Set("pair", pair)
	return query, nil
}

func normalizeFuturesAccountBalances(includeZero bool, response []futuresAccountBalanceResponse) ([]exchange.FuturesAccountBalance, error) {
	balances := make([]exchange.FuturesAccountBalance, 0, len(response))
	for _, item := range response {
		if !includeZero {
			nonZero, err := futuresAccountBalanceNonZero(item)
			if err != nil {
				return nil, err
			}
			if !nonZero {
				continue
			}
		}
		balances = append(balances, exchange.FuturesAccountBalance{
			AccountAlias:          item.AccountAlias,
			Asset:                 item.Asset,
			Balance:               item.Balance,
			WithdrawAvailable:     item.WithdrawAvailable,
			CrossWalletBalance:    item.CrossWalletBalance,
			CrossUnrealizedProfit: item.CrossUnrealizedProfit,
			AvailableBalance:      item.AvailableBalance,
			MaxWithdrawAmount:     item.MaxWithdrawAmount,
			MarginAvailable:       item.MarginAvailable,
			UpdateTime:            item.UpdateTime,
		})
	}
	return balances, nil
}

func futuresAccountBalanceNonZero(item futuresAccountBalanceResponse) (bool, error) {
	values := []struct {
		name  string
		value string
	}{
		{name: "balance", value: item.Balance},
		{name: "withdraw_available", value: item.WithdrawAvailable},
		{name: "cross_wallet_balance", value: item.CrossWalletBalance},
		{name: "cross_unrealized_profit", value: item.CrossUnrealizedProfit},
		{name: "available_balance", value: item.AvailableBalance},
		{name: "max_withdraw_amount", value: item.MaxWithdrawAmount},
	}
	for _, value := range values {
		if strings.TrimSpace(value.value) == "" {
			continue
		}
		amount, err := decimal.NewFromString(value.value)
		if err != nil {
			return false, fmt.Errorf("parse %s %s: %w", item.Asset, value.name, err)
		}
		if !amount.IsZero() {
			return true, nil
		}
	}
	return false, nil
}

func normalizeFuturesPositions(includeZero bool, response []futuresPositionResponse) []exchange.FuturesPosition {
	positions := make([]exchange.FuturesPosition, 0, len(response))
	for _, item := range response {
		if !includeZero && isZeroPositionAmount(item.PositionAmount) {
			continue
		}
		var autoAddMargin *bool
		if item.IsAutoAddMargin != nil {
			value := bool(*item.IsAutoAddMargin)
			autoAddMargin = &value
		}
		positions = append(positions, exchange.FuturesPosition{
			Symbol:                 item.Symbol,
			Pair:                   item.Pair,
			PositionSide:           item.PositionSide,
			PositionAmount:         item.PositionAmount,
			EntryPrice:             item.EntryPrice,
			BreakEvenPrice:         item.BreakEvenPrice,
			MarkPrice:              item.MarkPrice,
			UnrealizedProfit:       item.UnrealizedProfit,
			LiquidationPrice:       item.LiquidationPrice,
			Leverage:               item.Leverage,
			MarginType:             item.MarginType,
			IsolatedMargin:         item.IsolatedMargin,
			IsAutoAddMargin:        autoAddMargin,
			Notional:               item.Notional,
			NotionalValue:          item.NotionalValue,
			MarginAsset:            item.MarginAsset,
			IsolatedWallet:         item.IsolatedWallet,
			InitialMargin:          item.InitialMargin,
			MaintenanceMargin:      item.MaintenanceMargin,
			PositionInitialMargin:  item.PositionInitialMargin,
			OpenOrderInitialMargin: item.OpenOrderInitialMargin,
			MaxNotionalValue:       item.MaxNotionalValue,
			MaxQuantity:            item.MaxQuantity,
			BidNotional:            item.BidNotional,
			AskNotional:            item.AskNotional,
			ADL:                    item.ADL,
			UpdateTime:             item.UpdateTime,
		})
	}
	return positions
}

func isZeroPositionAmount(value string) bool {
	amount, err := decimal.NewFromString(strings.TrimSpace(value))
	return err == nil && amount.IsZero()
}
