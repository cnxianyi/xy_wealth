package summary

import (
	"context"
	"errors"
	"testing"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/bank"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
)

func TestServiceGetAllowsPartialResults(t *testing.T) {
	exchanges := []exchange.Provider{
		stubExchange{name: "working", balances: []asset.Balance{{Symbol: "BTC", Total: "1"}}},
		stubExchange{name: "broken", err: errors.New("upstream unavailable")},
	}
	banks := []bank.Provider{stubBank{name: "bank-a", accounts: []bank.Account{{Currency: "CNY", Balance: "100"}}}}

	result := New(exchanges, banks).Get(context.Background())

	if result.Exchanges[0].Status != "ok" || len(result.Exchanges[0].Balances) != 1 {
		t.Fatalf("working exchange result = %#v", result.Exchanges[0])
	}
	if result.Exchanges[1].Status != "error" || result.Exchanges[1].Error == "" {
		t.Fatalf("broken exchange result = %#v", result.Exchanges[1])
	}
	if result.Banks[0].Status != "ok" || len(result.Banks[0].Accounts) != 1 {
		t.Fatalf("bank result = %#v", result.Banks[0])
	}
}

func TestServiceGetCollectsExchangeProducts(t *testing.T) {
	result := New([]exchange.Provider{stubProductExchange{
		stubExchange{name: "multi", balances: []asset.Balance{{Symbol: "BTC", Total: "1"}}},
	}}, nil).Get(context.Background())

	if result.Exchanges[0].Status != "ok" || len(result.Exchanges[0].Products) != 4 {
		t.Fatalf("exchange product result = %#v", result.Exchanges[0])
	}
	products := result.Exchanges[0].Products
	if products[0].Product != "usdm" || len(products[0].FuturesBalances) != 1 || len(products[0].FuturesPositions) != 1 {
		t.Fatalf("USDⓈ-M product = %#v", products[0])
	}
	if products[1].Product != "usdcm" || products[2].Product != "coinm" || products[3].Product != "contract" {
		t.Fatalf("product order = %#v", products)
	}
	if len(products[3].ContractBalances) != 1 || len(products[3].ContractPositions) != 1 {
		t.Fatalf("Contract product = %#v", products[3])
	}
}

func TestServiceGetReportsProductFailuresIndependently(t *testing.T) {
	result := New([]exchange.Provider{stubPartialProductExchange{
		stubProductExchange{stubExchange{name: "partial", balances: []asset.Balance{{Symbol: "BTC", Total: "1"}}}},
	}}, nil).Get(context.Background())

	if result.Exchanges[0].Status != "partial" || result.Exchanges[0].Error == "" {
		t.Fatalf("partial exchange result = %#v", result.Exchanges[0])
	}
	for _, product := range result.Exchanges[0].Products {
		if product.Product == "usdcm" {
			if product.Status != "partial" || product.Error == "" || len(product.FuturesBalances) != 1 {
				t.Fatalf("partial USDC-M product = %#v", product)
			}
			return
		}
	}
	t.Fatal("summary did not include USDC-M product")
}

func TestServiceGetWithOptionsFiltersZeroValues(t *testing.T) {
	includeZero := false
	result := New(
		[]exchange.Provider{stubExchange{name: "filter", balances: []asset.Balance{{Symbol: "ZERO", Total: "0"}, {Symbol: "BTC", Total: "1"}}}},
		[]bank.Provider{stubBank{name: "bank", accounts: []bank.Account{{Currency: "CNY", Balance: "0"}, {Currency: "USD", Balance: "4"}}}},
	).GetWithOptions(context.Background(), Options{IncludeZero: &includeZero})
	if len(result.Exchanges[0].Balances) != 1 || result.Exchanges[0].Balances[0].Symbol != "BTC" || len(result.Banks[0].Accounts) != 1 || result.Banks[0].Accounts[0].Currency != "USD" {
		t.Fatalf("GetWithOptions zero filtering = %#v", result)
	}

	snapshot := Snapshot{
		Exchanges: []ExchangeData{{
			Balances: []asset.Balance{
				{Symbol: "ZERO", Free: "0", Locked: "0", Total: "0"},
				{Symbol: "BTC", Free: "1", Locked: "0", Total: "1"},
			},
			Products: []ProductData{{
				FuturesBalances:   []exchange.FuturesAccountBalance{{Asset: "ZERO", Balance: "0"}, {Asset: "USDT", Balance: "2"}},
				FuturesPositions:  []exchange.FuturesPosition{{Symbol: "ZERO", PositionAmount: "0"}, {Symbol: "BTCUSDT", PositionAmount: "1"}},
				ContractBalances:  []asset.Balance{{Symbol: "ZERO", Total: "0"}, {Symbol: "USDT", Total: "3"}},
				ContractPositions: []exchange.ContractPosition{{Symbol: "ZERO", Size: "0"}, {Symbol: "BTCUSDT", Size: "1"}},
			}},
		}},
		Banks: []BankData{{Accounts: []bank.Account{{Currency: "CNY", Balance: "0"}, {Currency: "USD", Balance: "4"}}}},
	}
	filterZeroValues(&snapshot)

	if len(snapshot.Exchanges[0].Balances) != 1 || snapshot.Exchanges[0].Balances[0].Symbol != "BTC" {
		t.Fatalf("spot zero filtering = %#v", snapshot.Exchanges[0].Balances)
	}
	product := snapshot.Exchanges[0].Products[0]
	if len(product.FuturesBalances) != 1 || len(product.FuturesPositions) != 1 || len(product.ContractBalances) != 1 || len(product.ContractPositions) != 1 {
		t.Fatalf("product zero filtering = %#v", product)
	}
	if len(snapshot.Banks[0].Accounts) != 1 || snapshot.Banks[0].Accounts[0].Currency != "USD" {
		t.Fatalf("bank zero filtering = %#v", snapshot.Banks[0].Accounts)
	}
}

type stubExchange struct {
	name     string
	balances []asset.Balance
	err      error
}

type stubProductExchange struct {
	stubExchange
}

type stubPartialProductExchange struct {
	stubProductExchange
}

func (stubPartialProductExchange) USDCMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return nil, errors.New("USDC-M positions unavailable")
}

func (s stubProductExchange) USDSMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "USDT", Balance: "10"}}, nil
}

func (s stubProductExchange) USDSMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCUSDT", PositionSide: "LONG"}}, nil
}

func (s stubProductExchange) USDCMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "USDC", Balance: "10"}}, nil
}

func (s stubProductExchange) USDCMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCPERP", PositionSide: "LONG"}}, nil
}

func (s stubProductExchange) COINMFuturesAccountBalances(context.Context) ([]exchange.FuturesAccountBalance, error) {
	return []exchange.FuturesAccountBalance{{Asset: "BTC", Balance: "1"}}, nil
}

func (s stubProductExchange) COINMFuturesPositions(context.Context, string) ([]exchange.FuturesPosition, error) {
	return []exchange.FuturesPosition{{Symbol: "BTCUSD_PERP", PositionSide: "LONG"}}, nil
}

func (s stubProductExchange) ContractBalances(context.Context) ([]asset.Balance, error) {
	return []asset.Balance{{Symbol: "USDT", Total: "20"}}, nil
}

func (s stubProductExchange) ContractPositions(context.Context, string) ([]exchange.ContractPosition, error) {
	return []exchange.ContractPosition{{Symbol: "BTCUSDT", Side: "long"}}, nil
}

func (s stubExchange) Name() string { return s.name }
func (s stubExchange) Balances(context.Context) ([]asset.Balance, error) {
	return s.balances, s.err
}

type stubBank struct {
	name     string
	accounts []bank.Account
	err      error
}

func (s stubBank) Name() string { return s.name }
func (s stubBank) Accounts(context.Context) ([]bank.Account, error) {
	return s.accounts, s.err
}
