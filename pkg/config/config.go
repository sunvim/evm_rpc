package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Chain       ChainConfig       `mapstructure:"chain"`
	Server      ServerConfig      `mapstructure:"server"`
	Storage     StorageConfig     `mapstructure:"storage"`
	Cache       CacheConfig       `mapstructure:"cache"`
	RateLimit   RateLimitConfig   `mapstructure:"ratelimit"`
	WorkerPools WorkerPoolsConfig `mapstructure:"worker_pools"`
	EVM         EVMConfig         `mapstructure:"evm"`
	API         APIConfig         `mapstructure:"api"`
	Metrics     MetricsConfig     `mapstructure:"metrics"`
	Logging     LoggingConfig     `mapstructure:"logging"`
}

type ChainConfig struct {
	Name      string `mapstructure:"name"`
	NetworkID uint64 `mapstructure:"network_id"`
	ChainID   uint64 `mapstructure:"chain_id"`
}

type ServerConfig struct {
	HTTP   HTTPConfig   `mapstructure:"http"`
	WS     WSConfig     `mapstructure:"ws"`
	Health HealthConfig `mapstructure:"health"`
}

type HTTPConfig struct {
	Enabled        bool          `mapstructure:"enabled"`
	ListenAddr     string        `mapstructure:"listen_addr"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
	IdleTimeout    time.Duration `mapstructure:"idle_timeout"`
	MaxHeaderBytes int           `mapstructure:"max_header_bytes"`
	CORSOrigins    []string      `mapstructure:"cors_origins"`
	VHosts         []string      `mapstructure:"vhosts"`
}

type WSConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	ListenAddr      string `mapstructure:"listen_addr"`
	MaxConnections  int    `mapstructure:"max_connections"`
	ReadBufferSize  int    `mapstructure:"read_buffer_size"`
	WriteBufferSize int    `mapstructure:"write_buffer_size"`
}

type HealthConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	ListenAddr string `mapstructure:"listen_addr"`
}

type StorageConfig struct {
	Pika PikaConfig `mapstructure:"pika"`
}

type PikaConfig struct {
	Addr           string        `mapstructure:"addr"`
	Password       string        `mapstructure:"password"`
	DB             int           `mapstructure:"db"`
	MaxConnections int           `mapstructure:"max_connections"`
	DialTimeout    time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout    time.Duration `mapstructure:"read_timeout"`
	WriteTimeout   time.Duration `mapstructure:"write_timeout"`
}

type CacheConfig struct {
	Enabled           bool               `mapstructure:"enabled"`
	BlockCacheSize    int                `mapstructure:"block_cache_size"`
	TxCacheSize       int                `mapstructure:"tx_cache_size"`
	ReceiptCacheSize  int                `mapstructure:"receipt_cache_size"`
	BalanceCacheSize  int                `mapstructure:"balance_cache_size"`
	CodeCacheSize     int                `mapstructure:"code_cache_size"`
	TTL               CacheTTLConfig     `mapstructure:"ttl"`
}

type CacheTTLConfig struct {
	Block       time.Duration `mapstructure:"block"`
	Transaction time.Duration `mapstructure:"transaction"`
	Receipt     time.Duration `mapstructure:"receipt"`
	Balance     time.Duration `mapstructure:"balance"`
	Code        time.Duration `mapstructure:"code"`
}

type RateLimitConfig struct {
	Enabled bool                       `mapstructure:"enabled"`
	Global  RateLimitRuleConfig        `mapstructure:"global"`
	IP      RateLimitRuleConfig        `mapstructure:"ip"`
	Method  map[string]int             `mapstructure:"method"`
}

type RateLimitRuleConfig struct {
	RequestsPerSecond int `mapstructure:"requests_per_second"`
	Burst             int `mapstructure:"burst"`
}

type WorkerPoolsConfig struct {
	Query   PoolConfig `mapstructure:"query"`
	Compute PoolConfig `mapstructure:"compute"`
	Write   PoolConfig `mapstructure:"write"`
}

type PoolConfig struct {
	WorkerCount int `mapstructure:"worker_count"`
	QueueSize   int `mapstructure:"queue_size"`
}

type EVMConfig struct {
	CallGasLimit         uint64  `mapstructure:"call_gas_limit"`
	EstimateGasMultiplier float64 `mapstructure:"estimate_gas_multiplier"`
}

type APIConfig struct {
	EnabledNamespaces []string `mapstructure:"enabled_namespaces"`
	DisabledMethods   []string `mapstructure:"disabled_methods"`
}

type MetricsConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	ListenAddr string `mapstructure:"listen_addr"`
}

type LoggingConfig struct {
	Level              string        `mapstructure:"level"`
	Format             string        `mapstructure:"format"`
	Output             string        `mapstructure:"output"`
	SlowQueryThreshold time.Duration `mapstructure:"slow_query_threshold"`
}

// LoadConfig loads configuration from file
func LoadConfig(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfigWithDefaults loads configuration with environment variable support
func LoadConfigWithDefaults(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}
