// Package auth implements the local secret login and opaque x-token sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/redis/go-redis/v9"
)

const (
	tokenBytes       = 32
	sessionKeyPrefix = "auth:session:"
)

var ErrInvalidSecret = errors.New("invalid authentication secret")

type Session struct {
	Token     string    `json:"x_token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Service struct {
	secret      [sha256.Size]byte
	tokenTTL    time.Duration
	redisClient *redis.Client
	random      io.Reader
	now         func() time.Time
}

func New(cfg config.AuthConfig, redisClient *redis.Client) *Service {
	return &Service{
		secret:      sha256.Sum256([]byte(cfg.Secret)),
		tokenTTL:    cfg.TokenTTL,
		redisClient: redisClient,
		random:      rand.Reader,
		now:         time.Now,
	}
}

func (s *Service) Login(ctx context.Context, candidate string) (Session, error) {
	candidateHash := sha256.Sum256([]byte(candidate))
	if subtle.ConstantTimeCompare(candidateHash[:], s.secret[:]) != 1 {
		return Session{}, ErrInvalidSecret
	}

	raw := make([]byte, tokenBytes)
	if _, err := io.ReadFull(s.random, raw); err != nil {
		return Session{}, fmt.Errorf("generate authentication token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	expiresAt := s.now().UTC().Add(s.tokenTTL)
	if err := s.redisClient.Set(ctx, sessionKey(token), expiresAt.Format(time.RFC3339Nano), s.tokenTTL).Err(); err != nil {
		return Session{}, fmt.Errorf("store authentication session: %w", err)
	}
	return Session{Token: token, ExpiresAt: expiresAt}, nil
}

func (s *Service) Validate(ctx context.Context, token string) (bool, error) {
	if !validTokenFormat(token) {
		return false, nil
	}
	exists, err := s.redisClient.Exists(ctx, sessionKey(token)).Result()
	if err != nil {
		return false, fmt.Errorf("validate authentication session: %w", err)
	}
	return exists == 1, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if !validTokenFormat(token) {
		return nil
	}
	if err := s.redisClient.Del(ctx, sessionKey(token)).Err(); err != nil {
		return fmt.Errorf("delete authentication session: %w", err)
	}
	return nil
}

func sessionKey(token string) string {
	digest := sha256.Sum256([]byte(token))
	return sessionKeyPrefix + hex.EncodeToString(digest[:])
}

func validTokenFormat(token string) bool {
	if token != strings.TrimSpace(token) || token == "" {
		return false
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	return err == nil && len(raw) == tokenBytes
}
