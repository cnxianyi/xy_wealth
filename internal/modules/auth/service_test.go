package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/cnxianyi/xy_wealth/internal/config"
	"github.com/redis/go-redis/v9"
)

func TestServiceSessionLifecycle(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })

	now := time.Date(2026, time.September, 3, 12, 0, 0, 0, time.UTC)
	service := New(config.AuthConfig{Secret: "correct-secret", TokenTTL: time.Hour}, client)
	service.now = func() time.Time { return now }
	service.random = bytes.NewReader(bytes.Repeat([]byte{0x2a}, tokenBytes))

	if _, err := service.Login(context.Background(), "wrong-secret"); err != ErrInvalidSecret {
		t.Fatalf("Login() error = %v, want ErrInvalidSecret", err)
	}

	session, err := service.Login(context.Background(), "correct-secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	wantToken := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{0x2a}, tokenBytes))
	if session.Token != wantToken || !session.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("session = %#v, want fixed token and one-hour expiry", session)
	}
	keys := server.Keys()
	if len(keys) != 1 || strings.Contains(keys[0], session.Token) {
		t.Fatalf("Redis keys = %#v, want one hashed session key", keys)
	}

	valid, err := service.Validate(context.Background(), session.Token)
	if err != nil || !valid {
		t.Fatalf("Validate() = %v, %v; want true, nil", valid, err)
	}
	if valid, err = service.Validate(context.Background(), "not-a-token"); err != nil || valid {
		t.Fatalf("Validate(malformed) = %v, %v; want false, nil", valid, err)
	}
	if err := service.Logout(context.Background(), session.Token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if valid, err = service.Validate(context.Background(), session.Token); err != nil || valid {
		t.Fatalf("Validate(after logout) = %v, %v; want false, nil", valid, err)
	}
}

func TestServiceSessionExpires(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	service := New(config.AuthConfig{Secret: "secret", TokenTTL: time.Minute}, client)

	session, err := service.Login(context.Background(), "secret")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	server.FastForward(time.Minute + time.Second)
	valid, err := service.Validate(context.Background(), session.Token)
	if err != nil || valid {
		t.Fatalf("Validate(expired) = %v, %v; want false, nil", valid, err)
	}
}
