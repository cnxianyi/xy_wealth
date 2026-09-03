package summary

import (
	"context"
	"errors"
	"testing"

	"github.com/xy-wealth/xy-wealth/internal/domain/asset"
	"github.com/xy-wealth/xy-wealth/internal/modules/bank"
	"github.com/xy-wealth/xy-wealth/internal/modules/exchange"
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

type stubExchange struct {
	name     string
	balances []asset.Balance
	err      error
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
