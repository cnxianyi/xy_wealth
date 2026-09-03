package summary

import (
	"context"
	"sync"
	"time"

	"github.com/xy-wealth/xy-wealth/internal/domain/asset"
	"github.com/xy-wealth/xy-wealth/internal/modules/bank"
	"github.com/xy-wealth/xy-wealth/internal/modules/exchange"
)

type Service struct {
	exchanges []exchange.Provider
	banks     []bank.Provider
}

type Snapshot struct {
	GeneratedAt time.Time      `json:"generated_at"`
	Exchanges   []ExchangeData `json:"exchanges"`
	Banks       []BankData     `json:"banks"`
}

type ExchangeData struct {
	Provider string          `json:"provider"`
	Status   string          `json:"status"`
	Balances []asset.Balance `json:"balances,omitempty"`
	Error    string          `json:"error,omitempty"`
}

type BankData struct {
	Provider string         `json:"provider"`
	Status   string         `json:"status"`
	Accounts []bank.Account `json:"accounts,omitempty"`
	Error    string         `json:"error,omitempty"`
}

func New(exchanges []exchange.Provider, banks []bank.Provider) *Service {
	return &Service{exchanges: exchanges, banks: banks}
}

func (s *Service) Get(ctx context.Context) Snapshot {
	result := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Exchanges:   make([]ExchangeData, len(s.exchanges)),
		Banks:       make([]BankData, len(s.banks)),
	}

	var wg sync.WaitGroup
	for i, provider := range s.exchanges {
		wg.Add(1)
		go func() {
			defer wg.Done()
			balances, err := provider.Balances(ctx)
			result.Exchanges[i] = ExchangeData{Provider: provider.Name(), Status: "ok", Balances: balances}
			if err != nil {
				result.Exchanges[i].Status = "error"
				result.Exchanges[i].Error = err.Error()
			}
		}()
	}
	for i, provider := range s.banks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			accounts, err := provider.Accounts(ctx)
			result.Banks[i] = BankData{Provider: provider.Name(), Status: "ok", Accounts: accounts}
			if err != nil {
				result.Banks[i].Status = "error"
				result.Banks[i].Error = err.Error()
			}
		}()
	}
	wg.Wait()
	return result
}
