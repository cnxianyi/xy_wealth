package bank

import "context"

type Account struct {
	AccountID string `json:"account_id"`
	Currency  string `json:"currency"`
	Balance   string `json:"balance"`
}

// Provider is implemented by each bank integration added in the future.
type Provider interface {
	Name() string
	Accounts(ctx context.Context) ([]Account, error)
}
