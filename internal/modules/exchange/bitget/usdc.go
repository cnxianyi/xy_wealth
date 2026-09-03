package bitget

import (
	"context"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

const bitgetUSDCFuturesProductType = "USDC-FUTURES"

var _ exchange.USDCMFuturesProvider = (*Client)(nil)
var _ exchange.USDCMFuturesAccountProvider = (*Client)(nil)

// USDCMFuturesAccountBalances returns Bitget's signed USDC-FUTURES account
// balances.
func (c *Client) USDCMFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	return c.futuresAccountBalances(ctx, bitgetUSDCFuturesProductType)
}

// USDCMFuturesPositions returns Bitget's signed USDC-FUTURES positions. The
// single-position endpoint requires the USDC margin coin.
func (c *Client) USDCMFuturesPositions(ctx context.Context, symbol string) ([]exchange.FuturesPosition, error) {
	marginCoin := ""
	if strings.TrimSpace(symbol) != "" {
		marginCoin = "USDC"
	}
	return c.futuresPositions(ctx, symbol, bitgetUSDCFuturesProductType, marginCoin)
}

// USDCMFuturesPing tests Bitget's public API for the USDC-FUTURES product.
func (c *Client) USDCMFuturesPing(ctx context.Context) error {
	return c.FuturesPing(ctx)
}

// USDCMFuturesServerTime returns Bitget's public server time.
func (c *Client) USDCMFuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	return c.ServerTime(ctx)
}

// USDCMFuturesExchangeInfo returns Bitget USDC-FUTURES contract configuration.
func (c *Client) USDCMFuturesExchangeInfo(ctx context.Context) (exchange.USDSMFuturesExchangeInfo, error) {
	response, err := c.contracts(ctx, bitgetUSDCFuturesProductType)
	if err != nil {
		return exchange.USDSMFuturesExchangeInfo{}, err
	}
	return normalizeFuturesExchangeInfo(response, bitgetUSDCFuturesProductType)
}

// USDCMFuturesDepth returns Bitget's merged USDC-FUTURES order book.
func (c *Client) USDCMFuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	return c.futuresDepth(ctx, symbol, limit, bitgetUSDCFuturesProductType)
}

// USDCMFuturesKlines returns Bitget USDC-FUTURES candlesticks.
func (c *Client) USDCMFuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	return c.futuresKlines(ctx, request, bitgetUSDCFuturesProductType)
}

// USDCMFuturesTicker24hr returns Bitget's USDC-FUTURES 24-hour ticker.
func (c *Client) USDCMFuturesTicker24hr(ctx context.Context, symbol string) (exchange.FuturesTicker24hr, error) {
	return c.futuresTicker24hr(ctx, symbol, bitgetUSDCFuturesProductType)
}

// USDCMFuturesTickerPrice returns Bitget's latest USDC-FUTURES price.
func (c *Client) USDCMFuturesTickerPrice(ctx context.Context, symbol string) (exchange.PriceTicker, error) {
	return c.futuresTickerPrice(ctx, symbol, bitgetUSDCFuturesProductType)
}

// USDCMFuturesBookTicker returns Bitget's best USDC-FUTURES bid and ask.
func (c *Client) USDCMFuturesBookTicker(ctx context.Context, symbol string) (exchange.BookTicker, error) {
	return c.futuresBookTicker(ctx, symbol, bitgetUSDCFuturesProductType)
}

// USDCMFuturesPremiumIndex returns Bitget's USDC-FUTURES mark/index/funding
// values.
func (c *Client) USDCMFuturesPremiumIndex(ctx context.Context, symbol string) (exchange.FuturesPremiumIndex, error) {
	return c.futuresPremiumIndex(ctx, symbol, bitgetUSDCFuturesProductType)
}
