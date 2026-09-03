package bitget

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

type futuresAccountResponse struct {
	MarginCoin          string       `json:"marginCoin"`
	Locked              numberString `json:"locked"`
	Available           numberString `json:"available"`
	CrossedMaxAvailable numberString `json:"crossedMaxAvailable"`
	MaxTransferOut      numberString `json:"maxTransferOut"`
	AccountEquity       numberString `json:"accountEquity"`
	UnrealizedPL        numberString `json:"unrealizedPL"`
}

type futuresPositionResponse struct {
	MarginCoin       string       `json:"marginCoin"`
	Symbol           string       `json:"symbol"`
	HoldSide         string       `json:"holdSide"`
	MarginSize       numberString `json:"marginSize"`
	Available        numberString `json:"available"`
	Locked           numberString `json:"locked"`
	Total            numberString `json:"total"`
	Leverage         numberString `json:"leverage"`
	OpenPriceAvg     numberString `json:"openPriceAvg"`
	MarginMode       string       `json:"marginMode"`
	UnrealizedPL     numberString `json:"unrealizedPL"`
	LiquidationPrice numberString `json:"liquidationPrice"`
	MarkPrice        numberString `json:"markPrice"`
	BreakEvenPrice   numberString `json:"breakEvenPrice"`
	UpdateTime       numberString `json:"uTime"`
	AutoMargin       string       `json:"autoMargin"`
}

var _ exchange.USDSMFuturesAccountProvider = (*Client)(nil)

// USDSMFuturesAccountBalances returns Bitget's signed USDT-FUTURES account
// balance. Bitget returns one account record per margin coin; the normalized
// model exposes the account equity and transfer limits as decimal strings.
func (c *Client) USDSMFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	return c.futuresAccountBalances(ctx, bitgetUSDTFuturesProductType)
}

func (c *Client) futuresAccountBalances(ctx context.Context, productType string) ([]exchange.FuturesAccountBalance, error) {
	query := url.Values{"productType": []string{productType}}
	var response []futuresAccountResponse
	if err := c.getSignedJSON(ctx, "/mix/account/accounts", query, &response); err != nil {
		return nil, err
	}
	balances := make([]exchange.FuturesAccountBalance, 0, len(response))
	for _, item := range response {
		nonZero, err := futuresAccountNonZero(item)
		if err != nil {
			return nil, err
		}
		if !c.includeZero && !nonZero {
			continue
		}
		balances = append(balances, exchange.FuturesAccountBalance{
			Asset:                 strings.ToUpper(strings.TrimSpace(item.MarginCoin)),
			Balance:               string(item.AccountEquity),
			WithdrawAvailable:     string(item.MaxTransferOut),
			CrossUnrealizedProfit: string(item.UnrealizedPL),
			AvailableBalance:      string(item.Available),
			MaxWithdrawAmount:     string(item.MaxTransferOut),
		})
	}
	return balances, nil
}

// USDSMFuturesPositions returns Bitget's signed current positions. An empty
// symbol uses all-position; a symbol uses the stricter single-position API.
func (c *Client) USDSMFuturesPositions(ctx context.Context, symbol string) ([]exchange.FuturesPosition, error) {
	return c.futuresPositions(ctx, symbol, bitgetUSDTFuturesProductType, "USDT")
}

func (c *Client) futuresPositions(ctx context.Context, symbol, productType, marginCoin string) ([]exchange.FuturesPosition, error) {
	normalized := strings.TrimSpace(symbol)
	path := "/mix/position/all-position"
	query := url.Values{"productType": []string{productType}}
	if normalized != "" {
		value, err := normalizeSymbol(normalized)
		if err != nil {
			return nil, err
		}
		path = "/mix/position/single-position"
		query.Set("symbol", value)
		if marginCoin != "" {
			query.Set("marginCoin", marginCoin)
		}
	}
	var response []futuresPositionResponse
	if err := c.getSignedJSON(ctx, path, query, &response); err != nil {
		return nil, err
	}
	positions := make([]exchange.FuturesPosition, 0, len(response))
	for _, item := range response {
		updateTime, err := parseOptionalInt(string(item.UpdateTime))
		if err != nil {
			return nil, fmt.Errorf("parse %s position update time: %w", item.Symbol, err)
		}
		autoMargin := strings.EqualFold(strings.TrimSpace(item.AutoMargin), "on")
		positionSide := strings.ToUpper(strings.TrimSpace(item.HoldSide))
		if positionSide != "LONG" && positionSide != "SHORT" {
			positionSide = "BOTH"
		}
		marginType := strings.ToLower(strings.TrimSpace(item.MarginMode))
		if marginType == "crossed" {
			marginType = "cross"
		}
		isolatedMargin := ""
		if marginType == "isolated" {
			isolatedMargin = string(item.MarginSize)
		}
		positions = append(positions, exchange.FuturesPosition{
			Symbol:                item.Symbol,
			Pair:                  item.Symbol,
			PositionSide:          positionSide,
			PositionAmount:        string(item.Total),
			EntryPrice:            string(item.OpenPriceAvg),
			BreakEvenPrice:        string(item.BreakEvenPrice),
			MarkPrice:             string(item.MarkPrice),
			UnrealizedProfit:      string(item.UnrealizedPL),
			LiquidationPrice:      string(item.LiquidationPrice),
			Leverage:              string(item.Leverage),
			MarginType:            marginType,
			IsolatedMargin:        isolatedMargin,
			IsAutoAddMargin:       &autoMargin,
			MarginAsset:           item.MarginCoin,
			PositionInitialMargin: string(item.MarginSize),
			UpdateTime:            updateTime,
		})
	}
	return positions, nil
}

func futuresAccountNonZero(item futuresAccountResponse) (bool, error) {
	values := []struct {
		name  string
		value string
	}{
		{name: "locked", value: string(item.Locked)},
		{name: "available", value: string(item.Available)},
		{name: "crossed_max_available", value: string(item.CrossedMaxAvailable)},
		{name: "max_transfer_out", value: string(item.MaxTransferOut)},
		{name: "account_equity", value: string(item.AccountEquity)},
		{name: "unrealized_pl", value: string(item.UnrealizedPL)},
	}
	for _, value := range values {
		if strings.TrimSpace(value.value) == "" {
			continue
		}
		amount, err := decimal.NewFromString(value.value)
		if err != nil {
			return false, fmt.Errorf("parse %s %s: %w", item.MarginCoin, value.name, err)
		}
		if !amount.IsZero() {
			return true, nil
		}
	}
	return false, nil
}
