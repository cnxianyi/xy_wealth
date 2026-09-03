package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Auth     AuthConfig     `mapstructure:"auth"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Web      WebConfig      `mapstructure:"web"`
	Log      LogConfig      `mapstructure:"log"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Binance  BinanceConfig  `mapstructure:"binance"`
	Bitget   BitgetConfig   `mapstructure:"bitget"`
	Weex     WeexConfig     `mapstructure:"weex"`
}

type AuthConfig struct {
	Secret   string        `mapstructure:"secret"`
	TokenTTL time.Duration `mapstructure:"token_ttl"`
}

type WebConfig struct {
	StaticDir string `mapstructure:"static_dir"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Environment string `mapstructure:"environment"`
}

type HTTPConfig struct {
	Address         string        `mapstructure:"address"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	IdleTimeout     time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

type LogConfig struct {
	Level    string `mapstructure:"level"`
	Encoding string `mapstructure:"encoding"`
}

type PostgresConfig struct {
	DSN             string        `mapstructure:"dsn"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
}

type RedisConfig struct {
	Address      string        `mapstructure:"address"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type BinanceConfig struct {
	BaseURL             string        `mapstructure:"base_url"`
	FuturesBaseURL      string        `mapstructure:"futures_base_url"`
	CoinMFuturesBaseURL string        `mapstructure:"coin_m_futures_base_url"`
	APIKey              string        `mapstructure:"api_key"`
	SecretKey           string        `mapstructure:"secret_key"`
	RecvWindow          int64         `mapstructure:"recv_window"`
	HTTPTimeout         time.Duration `mapstructure:"http_timeout"`
	IncludeZero         bool          `mapstructure:"include_zero"`
}

type BitgetConfig struct {
	BaseURL     string        `mapstructure:"base_url"`
	APIKey      string        `mapstructure:"api_key"`
	SecretKey   string        `mapstructure:"secret_key"`
	Passphrase  string        `mapstructure:"passphrase"`
	HTTPTimeout time.Duration `mapstructure:"http_timeout"`
	IncludeZero bool          `mapstructure:"include_zero"`
}

type WeexConfig struct {
	SpotBaseURL     string        `mapstructure:"spot_base_url"`
	ContractBaseURL string        `mapstructure:"contract_base_url"`
	APIKey          string        `mapstructure:"api_key"`
	SecretKey       string        `mapstructure:"secret_key"`
	Passphrase      string        `mapstructure:"passphrase"`
	HTTPTimeout     time.Duration `mapstructure:"http_timeout"`
	IncludeZero     bool          `mapstructure:"include_zero"`
}

func Load() (Config, error) {
	v := viper.New()
	setDefaults(v)

	if configFile := os.Getenv("XY_WEALTH_CONFIG_FILE"); configFile != "" {
		v.SetConfigFile(configFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath("./configs")
		v.AddConfigPath(".")
	}

	v.SetEnvPrefix("XY_WEALTH")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for _, key := range envKeys {
		if err := v.BindEnv(key); err != nil {
			return Config{}, fmt.Errorf("bind environment variable %q: %w", key, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return Config{}, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var missing []string
	if c.HTTP.Address == "" {
		missing = append(missing, "http.address")
	}
	if c.Auth.Secret == "" {
		missing = append(missing, "auth.secret")
	}
	if c.Auth.TokenTTL <= 0 {
		missing = append(missing, "auth.token_ttl")
	}
	if c.Postgres.DSN == "" {
		missing = append(missing, "postgres.dsn")
	}
	if c.Redis.Address == "" {
		missing = append(missing, "redis.address")
	}
	if c.Binance.BaseURL == "" {
		missing = append(missing, "binance.base_url")
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required config: %s", strings.Join(missing, ", "))
	}
	return nil
}

var envKeys = []string{
	"app.name", "app.environment",
	"auth.secret", "auth.token_ttl",
	"http.address", "http.read_timeout", "http.write_timeout", "http.idle_timeout", "http.shutdown_timeout",
	"web.static_dir",
	"log.level", "log.encoding",
	"postgres.dsn", "postgres.max_open_conns", "postgres.max_idle_conns", "postgres.conn_max_lifetime", "postgres.conn_max_idle_time",
	"redis.address", "redis.password", "redis.db", "redis.pool_size", "redis.dial_timeout", "redis.read_timeout", "redis.write_timeout",
	"binance.base_url", "binance.futures_base_url", "binance.coin_m_futures_base_url", "binance.api_key", "binance.secret_key", "binance.recv_window", "binance.http_timeout", "binance.include_zero",
	"bitget.base_url", "bitget.api_key", "bitget.secret_key", "bitget.passphrase", "bitget.http_timeout", "bitget.include_zero",
	"weex.spot_base_url", "weex.contract_base_url", "weex.api_key", "weex.secret_key", "weex.passphrase", "weex.http_timeout", "weex.include_zero",
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.name", "xy-wealth")
	v.SetDefault("app.environment", "development")
	v.SetDefault("auth.token_ttl", "24h")
	v.SetDefault("web.static_dir", "web/dist")
	v.SetDefault("http.address", ":8080")
	v.SetDefault("http.read_timeout", "10s")
	v.SetDefault("http.write_timeout", "30s")
	v.SetDefault("http.idle_timeout", "60s")
	v.SetDefault("http.shutdown_timeout", "10s")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.encoding", "json")
	v.SetDefault("postgres.dsn", "postgres://xy_wealth:xy_wealth@localhost:5432/xy_wealth?sslmode=disable")
	v.SetDefault("postgres.max_open_conns", 25)
	v.SetDefault("postgres.max_idle_conns", 5)
	v.SetDefault("postgres.conn_max_lifetime", "30m")
	v.SetDefault("postgres.conn_max_idle_time", "5m")
	v.SetDefault("redis.address", "localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("binance.base_url", "https://api.binance.com")
	v.SetDefault("binance.futures_base_url", "https://fapi.binance.com")
	v.SetDefault("binance.coin_m_futures_base_url", "https://dapi.binance.com")
	v.SetDefault("binance.recv_window", 5000)
	v.SetDefault("binance.http_timeout", "10s")
	v.SetDefault("binance.include_zero", false)
	v.SetDefault("bitget.base_url", "https://api.bitget.com")
	v.SetDefault("bitget.http_timeout", "10s")
	v.SetDefault("bitget.include_zero", false)
	v.SetDefault("weex.spot_base_url", "https://api-spot.weex.com")
	v.SetDefault("weex.contract_base_url", "https://api-contract.weex.com")
	v.SetDefault("weex.http_timeout", "10s")
	v.SetDefault("weex.include_zero", false)
}
