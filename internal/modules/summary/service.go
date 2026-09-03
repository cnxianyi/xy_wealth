package summary

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
	"github.com/cnxianyi/xy_wealth/internal/modules/bank"
	"github.com/cnxianyi/xy_wealth/internal/modules/exchange"
	"github.com/shopspring/decimal"
)

type Service struct {
	exchanges []exchange.Provider
	banks     []bank.Provider
}

// Options controls summary response filtering. A nil IncludeZero preserves
// the provider's original response; false removes zero-valued balances and
// positions, while true keeps them.
type Options struct {
	IncludeZero *bool
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
	Products []ProductData   `json:"products,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// ProductData is one independently collected account surface of an exchange.
// Futures products use FuturesBalances/FuturesPositions, while Weex Contract
// keeps its richer contract-specific balance and position models.
type ProductData struct {
	Product           string                           `json:"product"`
	Status            string                           `json:"status"`
	FuturesBalances   []exchange.FuturesAccountBalance `json:"futures_balances,omitempty"`
	FuturesPositions  []exchange.FuturesPosition       `json:"futures_positions,omitempty"`
	ContractBalances  []asset.Balance                  `json:"contract_balances,omitempty"`
	ContractPositions []exchange.ContractPosition      `json:"contract_positions,omitempty"`
	Error             string                           `json:"error,omitempty"`
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
	return s.GetWithOptions(ctx, Options{})
}

func (s *Service) GetWithOptions(ctx context.Context, options Options) Snapshot {
	result := Snapshot{
		GeneratedAt: time.Now().UTC(),
		Exchanges:   make([]ExchangeData, len(s.exchanges)),
		Banks:       make([]BankData, len(s.banks)),
	}

	var wg sync.WaitGroup
	for i, provider := range s.exchanges {
		wg.Add(1)
		go func(index int, provider exchange.Provider) {
			defer wg.Done()
			result.Exchanges[index] = s.collectExchange(ctx, provider)
		}(i, provider)
	}
	for i, provider := range s.banks {
		wg.Add(1)
		go func(index int, provider bank.Provider) {
			defer wg.Done()
			accounts, err := provider.Accounts(ctx)
			result.Banks[index] = BankData{Provider: provider.Name(), Status: "ok", Accounts: accounts}
			if err != nil {
				result.Banks[index].Status = "error"
				result.Banks[index].Error = err.Error()
			}
		}(i, provider)
	}
	wg.Wait()
	if options.IncludeZero != nil && !*options.IncludeZero {
		filterZeroValues(&result)
	}
	return result
}

func filterZeroValues(snapshot *Snapshot) {
	for exchangeIndex := range snapshot.Exchanges {
		item := &snapshot.Exchanges[exchangeIndex]
		item.Balances = filterSpotBalances(item.Balances)
		for productIndex := range item.Products {
			product := &item.Products[productIndex]
			product.FuturesBalances = filterFuturesBalances(product.FuturesBalances)
			product.FuturesPositions = filterFuturesPositions(product.FuturesPositions)
			product.ContractBalances = filterSpotBalances(product.ContractBalances)
			product.ContractPositions = filterContractPositions(product.ContractPositions)
		}
	}
	for bankIndex := range snapshot.Banks {
		snapshot.Banks[bankIndex].Accounts = filterBankAccounts(snapshot.Banks[bankIndex].Accounts)
	}
}

func filterSpotBalances(values []asset.Balance) []asset.Balance {
	filtered := make([]asset.Balance, 0, len(values))
	for _, value := range values {
		if !allZero(value.Free, value.Locked, value.Total) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func filterFuturesBalances(values []exchange.FuturesAccountBalance) []exchange.FuturesAccountBalance {
	filtered := make([]exchange.FuturesAccountBalance, 0, len(values))
	for _, value := range values {
		if !allZero(value.Balance, value.WithdrawAvailable, value.CrossWalletBalance, value.CrossUnrealizedProfit, value.AvailableBalance, value.MaxWithdrawAmount) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func filterFuturesPositions(values []exchange.FuturesPosition) []exchange.FuturesPosition {
	filtered := make([]exchange.FuturesPosition, 0, len(values))
	for _, value := range values {
		if !allZero(value.PositionAmount) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func filterContractPositions(values []exchange.ContractPosition) []exchange.ContractPosition {
	filtered := make([]exchange.ContractPosition, 0, len(values))
	for _, value := range values {
		if !allZero(value.Size) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func filterBankAccounts(values []bank.Account) []bank.Account {
	filtered := make([]bank.Account, 0, len(values))
	for _, value := range values {
		if !allZero(value.Balance) {
			filtered = append(filtered, value)
		}
	}
	return filtered
}

func allZero(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		amount, err := decimal.NewFromString(value)
		if err != nil || !amount.IsZero() {
			return false
		}
	}
	return true
}

func (s *Service) collectExchange(ctx context.Context, provider exchange.Provider) ExchangeData {
	data := ExchangeData{Provider: provider.Name(), Status: "ok"}
	balances, balanceErr := provider.Balances(ctx)
	data.Balances = balances

	if accountProvider, ok := provider.(exchange.USDSMFuturesAccountProvider); ok {
		data.Products = append(data.Products, collectFuturesProduct(ctx, "usdm", accountProvider))
	}
	if accountProvider, ok := provider.(exchange.USDCMFuturesAccountProvider); ok {
		data.Products = append(data.Products, collectFuturesProductUSDC(ctx, "usdcm", accountProvider))
	}
	if accountProvider, ok := provider.(exchange.COINMFuturesAccountProvider); ok {
		data.Products = append(data.Products, collectFuturesProductCOIN(ctx, "coinm", accountProvider))
	}
	if balanceProvider, balanceOK := provider.(exchange.ContractBalanceProvider); balanceOK {
		positionProvider, _ := provider.(exchange.ContractPositionProvider)
		data.Products = append(data.Products, collectContractProduct(ctx, balanceProvider, positionProvider))
	} else if positionProvider, positionOK := provider.(exchange.ContractPositionProvider); positionOK {
		data.Products = append(data.Products, collectContractProduct(ctx, nil, positionProvider))
	}

	errorsFound := make([]string, 0, len(data.Products)+1)
	hasSuccess := balanceErr == nil
	if balanceErr != nil {
		errorsFound = append(errorsFound, fmt.Sprintf("spot balances: %v", balanceErr))
	}
	for _, product := range data.Products {
		if product.Status == "ok" || product.Status == "partial" {
			hasSuccess = true
		}
		if product.Error != "" {
			errorsFound = append(errorsFound, fmt.Sprintf("%s: %s", product.Product, product.Error))
		}
	}
	switch {
	case len(errorsFound) == 0:
		data.Status = "ok"
	case !hasSuccess:
		data.Status = "error"
	default:
		data.Status = "partial"
	}
	if len(errorsFound) > 0 {
		data.Error = strings.Join(errorsFound, "; ")
	}
	return data
}

func collectFuturesProduct(ctx context.Context, product string, provider exchange.USDSMFuturesAccountProvider) ProductData {
	balances, balanceErr := provider.USDSMFuturesAccountBalances(ctx)
	positions, positionErr := provider.USDSMFuturesPositions(ctx, "")
	result := ProductData{Product: product, Status: "ok", FuturesBalances: balances, FuturesPositions: positions}
	return finishProduct(result, balanceErr, positionErr, 2)
}

func collectFuturesProductUSDC(ctx context.Context, product string, provider exchange.USDCMFuturesAccountProvider) ProductData {
	balances, balanceErr := provider.USDCMFuturesAccountBalances(ctx)
	positions, positionErr := provider.USDCMFuturesPositions(ctx, "")
	result := ProductData{Product: product, Status: "ok", FuturesBalances: balances, FuturesPositions: positions}
	return finishProduct(result, balanceErr, positionErr, 2)
}

func collectFuturesProductCOIN(ctx context.Context, product string, provider exchange.COINMFuturesAccountProvider) ProductData {
	balances, balanceErr := provider.COINMFuturesAccountBalances(ctx)
	positions, positionErr := provider.COINMFuturesPositions(ctx, "")
	result := ProductData{Product: product, Status: "ok", FuturesBalances: balances, FuturesPositions: positions}
	return finishProduct(result, balanceErr, positionErr, 2)
}

func collectContractProduct(ctx context.Context, balances exchange.ContractBalanceProvider, positions exchange.ContractPositionProvider) ProductData {
	result := ProductData{Product: "contract", Status: "ok"}
	var balanceErr, positionErr error
	operationCount := 0
	if balances != nil {
		operationCount++
		result.ContractBalances, balanceErr = balances.ContractBalances(ctx)
	}
	if positions != nil {
		operationCount++
		result.ContractPositions, positionErr = positions.ContractPositions(ctx, "")
	}
	return finishProduct(result, balanceErr, positionErr, operationCount)
}

func finishProduct(result ProductData, balanceErr, positionErr error, operationCount int) ProductData {
	errorsFound := make([]string, 0, 2)
	if balanceErr != nil {
		errorsFound = append(errorsFound, fmt.Sprintf("balances: %v", balanceErr))
	}
	if positionErr != nil {
		errorsFound = append(errorsFound, fmt.Sprintf("positions: %v", positionErr))
	}
	switch {
	case len(errorsFound) == 0:
		result.Status = "ok"
	case len(errorsFound) >= operationCount:
		result.Status = "error"
	default:
		result.Status = "partial"
	}
	if len(errorsFound) > 0 {
		result.Error = strings.Join(errorsFound, "; ")
	}
	return result
}
