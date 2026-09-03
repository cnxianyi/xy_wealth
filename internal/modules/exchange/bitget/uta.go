package bitget

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

// Bitget UTA exposes one account-assets response for all products. The
// product-specific position endpoint still accepts a category parameter.
type utaAssetsResponse struct {
	Assets []utaAssetResponse `json:"assets"`
}

type utaAssetResponse struct {
	Coin      string       `json:"coin"`
	Equity    numberString `json:"equity"`
	Balance   numberString `json:"balance"`
	Available numberString `json:"available"`
	Locked    numberString `json:"locked"`
}

type utaPositionsResponse struct {
	List []utaPositionResponse `json:"list"`
}

type utaPositionResponse struct {
	Category         string       `json:"category"`
	Symbol           string       `json:"symbol"`
	MarginCoin       string       `json:"marginCoin"`
	PosSide          string       `json:"posSide"`
	PositionBalance  numberString `json:"positionBalance"`
	Available        numberString `json:"available"`
	Frozen           numberString `json:"frozen"`
	Total            numberString `json:"total"`
	Leverage         numberString `json:"leverage"`
	AvgPrice         numberString `json:"avgPrice"`
	MarginMode       string       `json:"marginMode"`
	UnrealisedPnl    numberString `json:"unrealisedPnl"`
	LiquidationPrice numberString `json:"liquidationPrice"`
	MarkPrice        numberString `json:"markPrice"`
	BreakEvenPrice   numberString `json:"breakEvenPrice"`
	UpdatedTime      numberString `json:"updatedTime"`
}

func (c *Client) utaAssets(ctx context.Context) ([]utaAssetResponse, error) {
	var response utaAssetsResponse
	if err := c.getSignedV3JSON(ctx, "/account/assets", nil, &response); err != nil {
		return nil, err
	}
	return response.Assets, nil
}

func (c *Client) utaSpotBalances(ctx context.Context) ([]asset.Balance, error) {
	response, err := c.utaAssets(ctx)
	if err != nil {
		return nil, err
	}
	balances := make([]asset.Balance, 0, len(response))
	for _, item := range response {
		if !c.includeZero && allZeroNumbers(item.Balance, item.Available, item.Locked) {
			continue
		}
		balances = append(balances, asset.Balance{
			Symbol: strings.ToUpper(strings.TrimSpace(item.Coin)),
			Free:   string(item.Available),
			Locked: string(item.Locked),
			Total:  string(item.Balance),
		})
	}
	return balances, nil
}

func (c *Client) utaFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	response, err := c.utaAssets(ctx)
	if err != nil {
		return nil, err
	}
	balances := make([]exchange.FuturesAccountBalance, 0, len(response))
	for _, item := range response {
		if !c.includeZero && allZeroNumbers(item.Equity, item.Balance, item.Available, item.Locked) {
			continue
		}
		equity := item.Equity
		if strings.TrimSpace(string(equity)) == "" {
			equity = item.Balance
		}
		balances = append(balances, exchange.FuturesAccountBalance{
			Asset:             strings.ToUpper(strings.TrimSpace(item.Coin)),
			Balance:           string(equity),
			AvailableBalance:  string(item.Available),
			WithdrawAvailable: string(item.Available),
			MaxWithdrawAmount: string(item.Available),
		})
	}
	return balances, nil
}

func (c *Client) utaFuturesPositions(ctx context.Context, productType, symbol string) ([]exchange.FuturesPosition, error) {
	query := url.Values{"category": []string{productType}}
	if strings.TrimSpace(symbol) != "" {
		normalized, err := normalizeSymbol(symbol)
		if err != nil {
			return nil, err
		}
		query.Set("symbol", normalized)
	}
	var response utaPositionsResponse
	if err := c.getSignedV3JSON(ctx, "/position/current-position", query, &response); err != nil {
		return nil, err
	}
	positions := make([]exchange.FuturesPosition, 0, len(response.List))
	for _, item := range response.List {
		if !c.includeZero {
			total := strings.TrimSpace(string(item.Total))
			if total != "" {
				amount, err := decimal.NewFromString(total)
				if err != nil {
					return nil, fmt.Errorf("parse %s UTA position amount: %w", item.Symbol, err)
				}
				if amount.IsZero() {
					continue
				}
			}
		}
		updateTime, err := parseOptionalInt(string(item.UpdatedTime))
		if err != nil {
			return nil, fmt.Errorf("parse %s UTA position update time: %w", item.Symbol, err)
		}
		positionSide := strings.ToUpper(strings.TrimSpace(item.PosSide))
		if positionSide != "LONG" && positionSide != "SHORT" {
			positionSide = "BOTH"
		}
		marginType := strings.ToLower(strings.TrimSpace(item.MarginMode))
		if marginType == "crossed" {
			marginType = "cross"
		}
		isolatedMargin := ""
		if marginType == "isolated" {
			isolatedMargin = string(item.PositionBalance)
		}
		positions = append(positions, exchange.FuturesPosition{
			Symbol:                item.Symbol,
			Pair:                  item.Symbol,
			PositionSide:          positionSide,
			PositionAmount:        string(item.Total),
			EntryPrice:            string(item.AvgPrice),
			BreakEvenPrice:        string(item.BreakEvenPrice),
			MarkPrice:             string(item.MarkPrice),
			UnrealizedProfit:      string(item.UnrealisedPnl),
			LiquidationPrice:      string(item.LiquidationPrice),
			Leverage:              string(item.Leverage),
			MarginType:            marginType,
			IsolatedMargin:        isolatedMargin,
			MarginAsset:           item.MarginCoin,
			PositionInitialMargin: string(item.PositionBalance),
			UpdateTime:            updateTime,
		})
	}
	return positions, nil
}

func allZeroNumbers(values ...numberString) bool {
	for _, value := range values {
		if strings.TrimSpace(string(value)) == "" {
			continue
		}
		amount, err := decimal.NewFromString(string(value))
		if err != nil || !amount.IsZero() {
			return false
		}
	}
	return true
}
