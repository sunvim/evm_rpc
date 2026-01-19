package storage

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/redis/go-redis/v9"
)

// TxPoolStorage handles transaction pool operations
type TxPoolStorage struct {
	client *PikaClient
}

// NewTxPoolStorage creates a new transaction pool storage
func NewTxPoolStorage(client *PikaClient) *TxPoolStorage {
	return &TxPoolStorage{client: client}
}

// AddPendingTx adds a transaction to the pending pool
func (t *TxPoolStorage) AddPendingTx(ctx context.Context, tx *types.Transaction, source string) error {
	txHash := tx.Hash()
	
	// Encode transaction
	data, err := rlp.EncodeToBytes(tx)
	if err != nil {
		return fmt.Errorf("failed to encode transaction: %w", err)
	}

	// Store transaction
	txKey := fmt.Sprintf("pool:pending:%s", txHash.Hex())
	if err := t.client.Set(ctx, txKey, data, 0); err != nil {
		return err
	}

	// Get sender
	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return fmt.Errorf("failed to get sender: %w", err)
	}

	// Add to address index (sorted by nonce)
	addrKey := fmt.Sprintf("pool:addr:%s", from.Hex())
	if err := t.client.ZAdd(ctx, addrKey, redis.Z{
		Score:  float64(tx.Nonce()),
		Member: txHash.Hex(),
	}); err != nil {
		return err
	}

	// Add to price index (sorted by gas price)
	gasPrice := tx.GasPrice()
	if gasPrice == nil {
		gasPrice = tx.GasFeeCap()
	}
	
	if err := t.client.ZAdd(ctx, "pool:byprice", redis.Z{
		Score:  float64(gasPrice.Uint64()),
		Member: txHash.Hex(),
	}); err != nil {
		return err
	}

	// Publish to notification channel
	if err := t.client.Publish(ctx, "pool:new", txHash.Hex()); err != nil {
		return err
	}

	return nil
}

// GetPendingTx retrieves a pending transaction
func (t *TxPoolStorage) GetPendingTx(ctx context.Context, hash common.Hash) (*types.Transaction, error) {
	key := fmt.Sprintf("pool:pending:%s", hash.Hex())
	data, err := t.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %w", err)
	}

	return &tx, nil
}

// GetPendingTransactions returns all pending transactions
func (t *TxPoolStorage) GetPendingTransactions(ctx context.Context) (types.Transactions, error) {
	// Get all transaction hashes sorted by price (highest first)
	hashes, err := t.client.ZRevRange(ctx, "pool:byprice", 0, -1)
	if err != nil {
		return nil, err
	}

	var txs types.Transactions
	for _, hashStr := range hashes {
		hash := common.HexToHash(hashStr)
		tx, err := t.GetPendingTx(ctx, hash)
		if err != nil {
			continue // Skip failed transactions
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

// GetAddressTransactions returns pending transactions for an address
func (t *TxPoolStorage) GetAddressTransactions(ctx context.Context, address common.Address) (types.Transactions, error) {
	key := fmt.Sprintf("pool:addr:%s", address.Hex())
	hashes, err := t.client.ZRange(ctx, key, 0, -1)
	if err != nil {
		return nil, err
	}

	var txs types.Transactions
	for _, hashStr := range hashes {
		hash := common.HexToHash(hashStr)
		tx, err := t.GetPendingTx(ctx, hash)
		if err != nil {
			continue
		}
		txs = append(txs, tx)
	}

	return txs, nil
}

// RemovePendingTx removes a transaction from the pending pool
func (t *TxPoolStorage) RemovePendingTx(ctx context.Context, hash common.Hash) error {
	// Get transaction to find sender
	tx, err := t.GetPendingTx(ctx, hash)
	if err != nil {
		return err
	}

	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return err
	}

	// Remove from storage
	txKey := fmt.Sprintf("pool:pending:%s", hash.Hex())
	if err := t.client.Del(ctx, txKey); err != nil {
		return err
	}

	// Remove from address index
	addrKey := fmt.Sprintf("pool:addr:%s", from.Hex())
	if err := t.client.ZRem(ctx, addrKey, hash.Hex()); err != nil {
		return err
	}

	// Remove from price index
	if err := t.client.ZRem(ctx, "pool:byprice", hash.Hex()); err != nil {
		return err
	}

	return nil
}

// GetPoolStatus returns transaction pool statistics
func (t *TxPoolStorage) GetPoolStatus(ctx context.Context) (map[string]int, error) {
	pendingCount, err := t.client.ZCard(ctx, "pool:byprice")
	if err != nil {
		return nil, err
	}

	return map[string]int{
		"pending": int(pendingCount),
		"queued":  0, // Not implemented for simplicity
	}, nil
}

// GetPoolContent returns full transaction pool content
func (t *TxPoolStorage) GetPoolContent(ctx context.Context) (map[string]map[string]map[string]*types.Transaction, error) {
	// Get all pending transactions
	txs, err := t.GetPendingTransactions(ctx)
	if err != nil {
		return nil, err
	}

	// Group by address and nonce
	pending := make(map[string]map[string]*types.Transaction)
	
	signer := types.LatestSigner(nil)
	for _, tx := range txs {
		from, err := types.Sender(signer, tx)
		if err != nil {
			continue
		}

		addr := from.Hex()
		if pending[addr] == nil {
			pending[addr] = make(map[string]*types.Transaction)
		}
		
		nonce := strconv.FormatUint(tx.Nonce(), 10)
		pending[addr][nonce] = tx
	}

	return map[string]map[string]map[string]*types.Transaction{
		"pending": pending,
		"queued":  make(map[string]map[string]*types.Transaction), // Empty for simplicity
	}, nil
}
