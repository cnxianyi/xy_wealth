// Package weex contains the Weex exchange integration boundary.
//
// The V3 Spot and Contract REST clients will be implemented as separate
// capabilities on Client. This first scaffold keeps their base domains and
// credentials isolated without exposing incomplete endpoints.
package weex

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/cnxianyi/xy_wealth/internal/domain/asset"
)

var ErrNotImplemented = errors.New("weex provider is not initialized")

type Client struct {
	spotBaseURL     string
	contractBaseURL string
	apiKey          string
	secretKey       string
	passphrase      string
	httpClient      *http.Client
}

func New(cfg config.WeexConfig) *Client {
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	spotBaseURL := cfg.SpotBaseURL
	if spotBaseURL == "" {
		spotBaseURL = "https://api-spot.weex.com"
	}
	contractBaseURL := cfg.ContractBaseURL
	if contractBaseURL == "" {
		contractBaseURL = "https://api-contract.weex.com"
	}
	return &Client{
		spotBaseURL:     strings.TrimRight(spotBaseURL, "/"),
		contractBaseURL: strings.TrimRight(contractBaseURL, "/"),
		apiKey:          cfg.APIKey,
		secretKey:       cfg.SecretKey,
		passphrase:      cfg.Passphrase,
		httpClient:      &http.Client{Timeout: timeout},
	}
}

func (c *Client) Name() string { return "weex" }

// Balances is kept as an explicit provider boundary until Weex's account
// response is mapped to the shared asset model.
func (c *Client) Balances(context.Context) ([]asset.Balance, error) {
	return nil, ErrNotImplemented
}
