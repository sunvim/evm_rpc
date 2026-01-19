package storage

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidData  = errors.New("invalid data")
)

// BlockReader reads block data from Pika
type BlockReader struct {
	client *PikaClient
}

// NewBlockReader creates a new block reader
func NewBlockReader(client *PikaClient) *BlockReader {
	return &BlockReader{client: client}
}

// GetLatestBlockNumber returns the latest block number
func (r *BlockReader) GetLatestBlockNumber(ctx context.Context) (uint64, error) {
	data, err := r.client.Get(ctx, "idx:latest")
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(data), 10, 64)
}

// GetBlockNumberByHash returns block number by hash
func (r *BlockReader) GetBlockNumberByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	key := fmt.Sprintf("idx:blk:hash:%s", hash.Hex())
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(string(data), 10, 64)
}

// GetHeader returns block header by number
func (r *BlockReader) GetHeader(ctx context.Context, number uint64) (*types.Header, error) {
	key := fmt.Sprintf("blk:hdr:%d", number)
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var header types.Header
	if err := rlp.DecodeBytes(data, &header); err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	return &header, nil
}

// GetBlockBody returns block body by number
func (r *BlockReader) GetBlockBody(ctx context.Context, number uint64) (*types.Body, error) {
	key := fmt.Sprintf("blk:body:%d", number)
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var body types.Body
	if err := rlp.DecodeBytes(data, &body); err != nil {
		return nil, fmt.Errorf("failed to decode body: %w", err)
	}

	return &body, nil
}

// GetBlock returns full block by number
func (r *BlockReader) GetBlock(ctx context.Context, number uint64) (*types.Block, error) {
	header, err := r.GetHeader(ctx, number)
	if err != nil {
		return nil, err
	}

	body, err := r.GetBlockBody(ctx, number)
	if err != nil {
		return nil, err
	}

	return types.NewBlockWithHeader(header).WithBody(body.Transactions, body.Uncles), nil
}

// GetBlockByHash returns full block by hash
func (r *BlockReader) GetBlockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	number, err := r.GetBlockNumberByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return r.GetBlock(ctx, number)
}

// GetReceipts returns receipts for a block
func (r *BlockReader) GetReceipts(ctx context.Context, number uint64) (types.Receipts, error) {
	key := fmt.Sprintf("blk:rcpt:%d", number)
	data, err := r.client.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	var receipts types.Receipts
	if err := rlp.DecodeBytes(data, &receipts); err != nil {
		return nil, fmt.Errorf("failed to decode receipts: %w", err)
	}

	return receipts, nil
}

// GetTransactionCount returns the number of transactions in a block
func (r *BlockReader) GetTransactionCount(ctx context.Context, number uint64) (uint64, error) {
	body, err := r.GetBlockBody(ctx, number)
	if err != nil {
		return 0, err
	}
	return uint64(len(body.Transactions)), nil
}

// GetTransactionCountByHash returns the number of transactions in a block by hash
func (r *BlockReader) GetTransactionCountByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	number, err := r.GetBlockNumberByHash(ctx, hash)
	if err != nil {
		return 0, err
	}
	return r.GetTransactionCount(ctx, number)
}

// GetUncleCount returns uncle count (always 0 for BSC)
func (r *BlockReader) GetUncleCount(ctx context.Context, number uint64) (uint64, error) {
	// BSC doesn't have uncles
	return 0, nil
}

// GetUncleCountByHash returns uncle count by hash (always 0 for BSC)
func (r *BlockReader) GetUncleCountByHash(ctx context.Context, hash common.Hash) (uint64, error) {
	// BSC doesn't have uncles
	return 0, nil
}
