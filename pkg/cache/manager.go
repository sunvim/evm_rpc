package cache

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sunvim/evm_rpc/pkg/config"
)

// Manager manages multiple caches for different data types
type Manager struct {
	blockCache   *Cache
	txCache      *Cache
	receiptCache *Cache
	balanceCache *Cache
	codeCache    *Cache
	
	ttl config.CacheTTLConfig
}

// NewManager creates a new cache manager
func NewManager(cfg config.CacheConfig) (*Manager, error) {
	blockCache, err := NewCache(cfg.BlockCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create block cache: %w", err)
	}

	txCache, err := NewCache(cfg.TxCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create tx cache: %w", err)
	}

	receiptCache, err := NewCache(cfg.ReceiptCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create receipt cache: %w", err)
	}

	balanceCache, err := NewCache(cfg.BalanceCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create balance cache: %w", err)
	}

	codeCache, err := NewCache(cfg.CodeCacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create code cache: %w", err)
	}

	return &Manager{
		blockCache:   blockCache,
		txCache:      txCache,
		receiptCache: receiptCache,
		balanceCache: balanceCache,
		codeCache:    codeCache,
		ttl:          cfg.TTL,
	}, nil
}

// Block cache methods

func (m *Manager) GetBlock(number uint64) (*types.Block, bool) {
	key := fmt.Sprintf("blk:%d", number)
	val, ok := m.blockCache.Get(key)
	if !ok {
		return nil, false
	}
	return val.(*types.Block), true
}

func (m *Manager) SetBlock(number uint64, block *types.Block) {
	key := fmt.Sprintf("blk:%d", number)
	m.blockCache.Set(key, block, m.ttl.Block)
}

func (m *Manager) GetBlockByHash(hash common.Hash) (*types.Block, bool) {
	key := fmt.Sprintf("blk:hash:%s", hash.Hex())
	val, ok := m.blockCache.Get(key)
	if !ok {
		return nil, false
	}
	return val.(*types.Block), true
}

func (m *Manager) SetBlockByHash(hash common.Hash, block *types.Block) {
	key := fmt.Sprintf("blk:hash:%s", hash.Hex())
	m.blockCache.Set(key, block, m.ttl.Block)
}

// Transaction cache methods

func (m *Manager) GetTransaction(hash common.Hash) (*types.Transaction, bool) {
	key := fmt.Sprintf("tx:%s", hash.Hex())
	val, ok := m.txCache.Get(key)
	if !ok {
		return nil, false
	}
	return val.(*types.Transaction), true
}

func (m *Manager) SetTransaction(hash common.Hash, tx *types.Transaction) {
	key := fmt.Sprintf("tx:%s", hash.Hex())
	m.txCache.Set(key, tx, m.ttl.Transaction)
}

// Receipt cache methods

func (m *Manager) GetReceipt(hash common.Hash) (*types.Receipt, bool) {
	key := fmt.Sprintf("rcpt:%s", hash.Hex())
	val, ok := m.receiptCache.Get(key)
	if !ok {
		return nil, false
	}
	return val.(*types.Receipt), true
}

func (m *Manager) SetReceipt(hash common.Hash, receipt *types.Receipt) {
	key := fmt.Sprintf("rcpt:%s", hash.Hex())
	m.receiptCache.Set(key, receipt, m.ttl.Receipt)
}

// Balance cache methods

func (m *Manager) GetBalance(address common.Address, blockNumber string) (interface{}, bool) {
	key := fmt.Sprintf("bal:%s:%s", address.Hex(), blockNumber)
	return m.balanceCache.Get(key)
}

func (m *Manager) SetBalance(address common.Address, blockNumber string, balance interface{}) {
	key := fmt.Sprintf("bal:%s:%s", address.Hex(), blockNumber)
	m.balanceCache.Set(key, balance, m.ttl.Balance)
}

// Code cache methods

func (m *Manager) GetCode(address common.Address) ([]byte, bool) {
	key := fmt.Sprintf("code:%s", address.Hex())
	val, ok := m.codeCache.Get(key)
	if !ok {
		return nil, false
	}
	return val.([]byte), true
}

func (m *Manager) SetCode(address common.Address, code []byte) {
	key := fmt.Sprintf("code:%s", address.Hex())
	m.codeCache.Set(key, code, m.ttl.Code)
}

// Stats returns statistics for all caches
func (m *Manager) Stats() map[string]CacheStats {
	return map[string]CacheStats{
		"block":   m.blockCache.Stats(),
		"tx":      m.txCache.Stats(),
		"receipt": m.receiptCache.Stats(),
		"balance": m.balanceCache.Stats(),
		"code":    m.codeCache.Stats(),
	}
}

// HitRate returns overall hit rate
func (m *Manager) HitRate() float64 {
	var totalHits, totalMisses uint64
	
	stats := m.Stats()
	for _, s := range stats {
		totalHits += s.Hits
		totalMisses += s.Misses
	}
	
	total := totalHits + totalMisses
	if total == 0 {
		return 0
	}
	
	return float64(totalHits) / float64(total)
}

// Clear clears all caches
func (m *Manager) Clear() {
	m.blockCache.Clear()
	m.txCache.Clear()
	m.receiptCache.Clear()
	m.balanceCache.Clear()
	m.codeCache.Clear()
}
