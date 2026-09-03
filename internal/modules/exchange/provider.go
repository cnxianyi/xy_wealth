package exchange

import (
	"context"

	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
)

// Provider is the stable boundary implemented by every exchange integration.
type Provider interface {
	Name() string
	Balances(ctx context.Context) ([]asset.Balance, error)
}
