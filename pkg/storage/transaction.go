package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// TransactionReader reads transaction data from Pika
type TransactionReader struct {
	client *PikaClient
}

// NewTransactionReader creates a new transaction reader
func NewTransactionReader(client *PikaClient) *TransactionReader {
	return &TransactionReader{client: client}
}

// TxLookup contains transaction location information
type TxLookup struct {
	BlockNumber uint64 `json:"blockNumber"`
	BlockHash   string `json:"blockHash"`
	Index       uint64 `json:"index"`
}

// GetTransaction returns transaction by hash
func (r *TransactionReader) GetTransaction(ctx context.Context, hash common.Hash) (*types.Transaction, error) {
	key := fmt.Sprintf("tx:%s", hash.Hex())
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var tx types.Transaction
	if err := rlp.DecodeBytes(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %w", err)
	}

	return &tx, nil
}

// GetTransactionLookup returns transaction lookup information
func (r *TransactionReader) GetTransactionLookup(ctx context.Context, hash common.Hash) (*TxLookup, error) {
	key := fmt.Sprintf("tx:lookup:%s", hash.Hex())
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var lookup TxLookup
	if err := json.Unmarshal(data, &lookup); err != nil {
		return nil, fmt.Errorf("failed to decode lookup: %w", err)
	}

	return &lookup, nil
}

// GetReceipt returns transaction receipt by hash
func (r *TransactionReader) GetReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, *TxLookup, error) {
	// Get lookup info first
	lookup, err := r.GetTransactionLookup(ctx, hash)
	if err != nil {
		return nil, nil, err
	}

	// Get all receipts for the block
	receiptsKey := fmt.Sprintf("blk:rcpt:%d", lookup.BlockNumber)
	receiptsData, err := r.client.Get(ctx, receiptsKey)
	if err != nil {
		return nil, nil, err
	}

	var receipts types.Receipts
	if err := rlp.DecodeBytes(receiptsData, &receipts); err != nil {
		return nil, nil, fmt.Errorf("failed to decode receipts: %w", err)
	}

	if lookup.Index >= uint64(len(receipts)) {
		return nil, nil, ErrNotFound
	}

	return receipts[lookup.Index], lookup, nil
}

// GetTransactionByBlockNumberAndIndex returns transaction by block number and index
func (r *TransactionReader) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNumber, index uint64) (*types.Transaction, error) {
	bodyKey := fmt.Sprintf("blk:body:%d", blockNumber)
	bodyData, err := r.client.Get(ctx, bodyKey)
	if err != nil {
		return nil, err
	}

	var body types.Body
	if err := rlp.DecodeBytes(bodyData, &body); err != nil {
		return nil, fmt.Errorf("failed to decode body: %w", err)
	}

	if index >= uint64(len(body.Transactions)) {
		return nil, ErrNotFound
	}

	return body.Transactions[index], nil
}

// GetTransactionByBlockHashAndIndex returns transaction by block hash and index
func (r *TransactionReader) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index uint64) (*types.Transaction, error) {
	// Get block number from hash
	numberKey := fmt.Sprintf("idx:blk:hash:%s", blockHash.Hex())
	numberData, err := r.client.Get(ctx, numberKey)
	if err != nil {
		return nil, err
	}

	var blockNumber uint64
	if _, err := fmt.Sscanf(string(numberData), "%d", &blockNumber); err != nil {
		return nil, fmt.Errorf("invalid block number: %w", err)
	}

	return r.GetTransactionByBlockNumberAndIndex(ctx, blockNumber, index)
}
