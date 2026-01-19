package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sunvim/evm_rpc/pkg/config"
)

// PikaClient wraps Redis client for Pika storage
type PikaClient struct {
	client *redis.Client
}

// NewPikaClient creates a new Pika client
func NewPikaClient(cfg config.PikaConfig) (*PikaClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.MaxConnections,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Pika: %w", err)
	}

	return &PikaClient{
		client: client,
	}, nil
}

// Get retrieves a value by key
func (p *PikaClient) Get(ctx context.Context, key string) ([]byte, error) {
	result, err := p.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	return result, err
}

// Set stores a value with key
func (p *PikaClient) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return p.client.Set(ctx, key, value, ttl).Err()
}

// MGet retrieves multiple values by keys
func (p *PikaClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return p.client.MGet(ctx, keys...).Result()
}

// HGet retrieves a field value from hash
func (p *PikaClient) HGet(ctx context.Context, key, field string) ([]byte, error) {
	result, err := p.client.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	return result, err
}

// HSet stores a field value in hash
func (p *PikaClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	return p.client.HSet(ctx, key, values...).Err()
}

// HGetAll retrieves all fields from hash
func (p *PikaClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return p.client.HGetAll(ctx, key).Result()
}

// ZAdd adds member to sorted set
func (p *PikaClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return p.client.ZAdd(ctx, key, members...).Err()
}

// ZRange retrieves members from sorted set by range
func (p *PikaClient) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return p.client.ZRange(ctx, key, start, stop).Result()
}

// ZRevRange retrieves members from sorted set in reverse order
func (p *PikaClient) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return p.client.ZRevRange(ctx, key, start, stop).Result()
}

// ZCard returns the cardinality of sorted set
func (p *PikaClient) ZCard(ctx context.Context, key string) (int64, error) {
	return p.client.ZCard(ctx, key).Result()
}

// ZRem removes members from sorted set
func (p *PikaClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return p.client.ZRem(ctx, key, members...).Err()
}

// SAdd adds members to set
func (p *PikaClient) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return p.client.SAdd(ctx, key, members...).Err()
}

// SMembers retrieves all members from set
func (p *PikaClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return p.client.SMembers(ctx, key).Result()
}

// SCard returns the cardinality of set
func (p *PikaClient) SCard(ctx context.Context, key string) (int64, error) {
	return p.client.SCard(ctx, key).Result()
}

// Del deletes keys
func (p *PikaClient) Del(ctx context.Context, keys ...string) error {
	return p.client.Del(ctx, keys...).Err()
}

// Exists checks if keys exist
func (p *PikaClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return p.client.Exists(ctx, keys...).Result()
}

// Subscribe subscribes to channels
func (p *PikaClient) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	return p.client.Subscribe(ctx, channels...)
}

// Publish publishes message to channel
func (p *PikaClient) Publish(ctx context.Context, channel string, message interface{}) error {
	return p.client.Publish(ctx, channel, message).Err()
}

// Pipeline creates a pipeline
func (p *PikaClient) Pipeline() redis.Pipeliner {
	return p.client.Pipeline()
}

// Close closes the client connection
func (p *PikaClient) Close() error {
	return p.client.Close()
}

// GetClient returns the underlying Redis client
func (p *PikaClient) GetClient() *redis.Client {
	return p.client
}
