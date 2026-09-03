package bitget

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

const bitgetCOINFuturesProductType = "COIN-FUTURES"

var _ exchange.COINMFuturesProvider = (*Client)(nil)
var _ exchange.COINMFuturesAccountProvider = (*Client)(nil)

// COINMFuturesAccountBalances returns Bitget's signed COIN-FUTURES account
// balance.
func (c *Client) COINMFuturesAccountBalances(ctx context.Context) ([]exchange.FuturesAccountBalance, error) {
	return c.futuresAccountBalances(ctx, bitgetCOINFuturesProductType)
}

// COINMFuturesPositions returns Bitget's signed COIN-FUTURES positions. The
// single-position endpoint requires a margin coin, derived from symbols such
// as BTCUSD_PERP or BTCUSD_240628.
func (c *Client) COINMFuturesPositions(ctx context.Context, symbol string) ([]exchange.FuturesPosition, error) {
	marginCoin := ""
	if strings.TrimSpace(symbol) != "" {
		value, err := normalizeSymbol(symbol)
		if err != nil {
			return nil, err
		}
		marginCoin = coinMarginCoin(value)
	}
	return c.futuresPositions(ctx, symbol, bitgetCOINFuturesProductType, marginCoin)
}

// CoinMFuturesPing tests Bitget's public API for the COIN-FUTURES product.
func (c *Client) CoinMFuturesPing(ctx context.Context) error {
	return c.FuturesPing(ctx)
}

// CoinMFuturesServerTime returns Bitget's public server time.
func (c *Client) CoinMFuturesServerTime(ctx context.Context) (exchange.ServerTime, error) {
	return c.ServerTime(ctx)
}

// CoinMFuturesExchangeInfo returns Bitget COIN-FUTURES contract configuration.
func (c *Client) CoinMFuturesExchangeInfo(ctx context.Context) (exchange.COINMFuturesExchangeInfo, error) {
	response, err := c.contracts(ctx, bitgetCOINFuturesProductType)
	if err != nil {
		return exchange.COINMFuturesExchangeInfo{}, err
	}
	info := exchange.COINMFuturesExchangeInfo{Timezone: "UTC", Symbols: make([]exchange.COINMFuturesSymbolInfo, 0, len(response))}
	for _, item := range response {
		deliveryDate, err := parseOptionalInt(string(item.DeliveryTime))
		if err != nil {
			return exchange.COINMFuturesExchangeInfo{}, fmt.Errorf("parse %s delivery time: %w", item.Symbol, err)
		}
		onboardDate, err := parseOptionalInt(string(item.LaunchTime))
		if err != nil {
			return exchange.COINMFuturesExchangeInfo{}, fmt.Errorf("parse %s launch time: %w", item.Symbol, err)
		}
		marginAsset := item.QuoteCoin
		if len(item.SupportMarginCoins) > 0 {
			marginAsset = item.SupportMarginCoins[0]
		}
		info.Symbols = append(info.Symbols, exchange.COINMFuturesSymbolInfo{
			Symbol:                item.Symbol,
			Pair:                  item.Symbol,
			ContractType:          strings.ToUpper(item.SymbolType),
			DeliveryDate:          deliveryDate,
			OnboardDate:           onboardDate,
			ContractStatus:        normalizeContractStatus(item.SymbolStatus),
			BaseAsset:             item.BaseCoin,
			QuoteAsset:            item.QuoteCoin,
			MarginAsset:           marginAsset,
			PricePrecision:        int(item.PricePlace),
			QuantityPrecision:     int(item.VolumePlace),
			BaseAssetPrecision:    int(item.VolumePlace),
			QuotePrecision:        int(item.PricePlace),
			MaintMarginPercent:    "",
			RequiredMarginPercent: "",
		})
	}
	return info, nil
}

// CoinMFuturesDepth returns Bitget's merged COIN-FUTURES order book.
func (c *Client) CoinMFuturesDepth(ctx context.Context, symbol string, limit int) (exchange.FuturesOrderBook, error) {
	return c.futuresDepth(ctx, symbol, limit, bitgetCOINFuturesProductType)
}

// CoinMFuturesKlines returns Bitget COIN-FUTURES candlesticks.
func (c *Client) CoinMFuturesKlines(ctx context.Context, request exchange.KlinesRequest) ([]exchange.Kline, error) {
	return c.futuresKlines(ctx, request, bitgetCOINFuturesProductType)
}

// CoinMFuturesTicker24hr returns Bitget's COIN-FUTURES 24-hour ticker.
func (c *Client) CoinMFuturesTicker24hr(ctx context.Context, symbol string) (exchange.COINMFuturesTicker24hr, error) {
	ticker, err := c.contractTicker(ctx, symbol, bitgetCOINFuturesProductType)
	if err != nil {
		return exchange.COINMFuturesTicker24hr{}, err
	}
	priceChange, weightedAverage, err := normalizedTickerValues(ticker)
	if err != nil {
		return exchange.COINMFuturesTicker24hr{}, err
	}
	changePercent, err := tickerChangePercent(ticker, priceChange)
	if err != nil {
		return exchange.COINMFuturesTicker24hr{}, err
	}
	timestamp, err := parseOptionalInt(string(ticker.Time))
	if err != nil {
		return exchange.COINMFuturesTicker24hr{}, fmt.Errorf("parse COIN-FUTURES ticker timestamp: %w", err)
	}
	return exchange.COINMFuturesTicker24hr{
		Symbol:             ticker.Symbol,
		Pair:               ticker.Symbol,
		PriceChange:        priceChange.String(),
		PriceChangePercent: changePercent,
		WeightedAvgPrice:   weightedAverage,
		LastPrice:          string(ticker.LastPrice),
		OpenPrice:          string(ticker.Open24h),
		HighPrice:          string(ticker.High24h),
		LowPrice:           string(ticker.Low24h),
		Volume:             string(ticker.BaseVolume),
		BaseVolume:         string(ticker.BaseVolume),
		OpenTime:           timestamp - 24*60*60*1000,
		CloseTime:          timestamp,
	}, nil
}

func (c *Client) CoinMFuturesTickerPrice(ctx context.Context, symbol string) (exchange.COINMFuturesPriceTicker, error) {
	normalized, err := normalizeSymbol(symbol)
	if err != nil {
		return exchange.COINMFuturesPriceTicker{}, err
	}
	query := url.Values{"productType": []string{bitgetCOINFuturesProductType}, "symbol": []string{normalized}}
	var response []contractPriceResponse
	if err := c.getJSON(ctx, "/mix/market/symbol-price", query, &response); err != nil {
		return exchange.COINMFuturesPriceTicker{}, err
	}
	if len(response) == 0 {
		return exchange.COINMFuturesPriceTicker{}, errors.New("Bitget COIN-FUTURES price response is empty")
	}
	timestamp, err := parseOptionalInt(string(response[0].Time))
	if err != nil {
		return exchange.COINMFuturesPriceTicker{}, fmt.Errorf("parse COIN-FUTURES price timestamp: %w", err)
	}
	return exchange.COINMFuturesPriceTicker{Symbol: response[0].Symbol, Pair: response[0].Symbol, Price: string(response[0].Price), Time: timestamp}, nil
}

func (c *Client) CoinMFuturesBookTicker(ctx context.Context, symbol string) (exchange.COINMFuturesBookTicker, error) {
	ticker, err := c.contractTicker(ctx, symbol, bitgetCOINFuturesProductType)
	if err != nil {
		return exchange.COINMFuturesBookTicker{}, err
	}
	timestamp, err := parseOptionalInt(string(ticker.Time))
	if err != nil {
		return exchange.COINMFuturesBookTicker{}, fmt.Errorf("parse COIN-FUTURES ticker timestamp: %w", err)
	}
	return exchange.COINMFuturesBookTicker{
		Symbol: ticker.Symbol, Pair: ticker.Symbol, BidPrice: string(ticker.BidPrice), BidQty: string(ticker.BidQty),
		AskPrice: string(ticker.AskPrice), AskQty: string(ticker.AskQty), Time: timestamp,
	}, nil
}

func (c *Client) CoinMFuturesPremiumIndex(ctx context.Context, symbol string) (exchange.COINMFuturesPremiumIndex, error) {
	ticker, err := c.contractTicker(ctx, symbol, bitgetCOINFuturesProductType)
	if err != nil {
		return exchange.COINMFuturesPremiumIndex{}, err
	}
	timestamp, err := parseOptionalInt(string(ticker.Time))
	if err != nil {
		return exchange.COINMFuturesPremiumIndex{}, fmt.Errorf("parse COIN-FUTURES ticker timestamp: %w", err)
	}
	return exchange.COINMFuturesPremiumIndex{
		Symbol: ticker.Symbol, Pair: ticker.Symbol, MarkPrice: string(ticker.MarkPrice), IndexPrice: string(ticker.IndexPrice),
		LastFundingRate: string(ticker.FundingRate), Time: timestamp,
	}, nil
}

func coinMarginCoin(symbol string) string {
	pair := symbol
	if separator := strings.IndexByte(pair, '_'); separator > 0 {
		pair = pair[:separator]
	}
	if strings.HasSuffix(pair, "USD") {
		pair = strings.TrimSuffix(pair, "USD")
	}
	return pair
}

func normalizedTickerValues(ticker contractTickerResponse) (priceChange decimal.Decimal, weightedAverage string, err error) {
	last, err := decimal.NewFromString(string(ticker.LastPrice))
	if err != nil {
		return decimal.Decimal{}, "", fmt.Errorf("parse ticker last price: %w", err)
	}
	open, err := decimal.NewFromString(string(ticker.Open24h))
	if err != nil {
		return decimal.Decimal{}, "", fmt.Errorf("parse ticker open price: %w", err)
	}
	priceChange = last.Sub(open)
	weighted, err := weightedAveragePrice(string(ticker.BaseVolume), string(ticker.QuoteVolume))
	if err != nil {
		return decimal.Decimal{}, "", err
	}
	weightedAverage = weighted
	return priceChange, weightedAverage, nil
}

func tickerChangePercent(ticker contractTickerResponse, change decimal.Decimal) (string, error) {
	if ticker.Change24h != "" {
		value, err := decimal.NewFromString(string(ticker.Change24h))
		if err != nil {
			return "", fmt.Errorf("parse COIN-FUTURES ticker 24-hour change: %w", err)
		}
		return value.Mul(decimal.NewFromInt(100)).String(), nil
	}
	open, err := decimal.NewFromString(string(ticker.Open24h))
	if err != nil {
		return "", fmt.Errorf("parse COIN-FUTURES ticker open price: %w", err)
	}
	if open.IsZero() {
		return "", nil
	}
	return change.Div(open).Mul(decimal.NewFromInt(100)).String(), nil
}
