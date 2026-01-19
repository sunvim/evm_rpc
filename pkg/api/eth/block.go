package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// BlockAPI provides block-related RPC methods
type BlockAPI struct {
	blockReader *storage.BlockReader
	chainID     uint64
}

// NewBlockAPI creates a new BlockAPI
func NewBlockAPI(blockReader *storage.BlockReader, chainID uint64) *BlockAPI {
	return &BlockAPI{
		blockReader: blockReader,
		chainID:     chainID,
	}
}

// resolveBlockNumber resolves a block number tag to actual block number
func (a *BlockAPI) resolveBlockNumber(ctx context.Context, blockNr api.BlockNumber) (uint64, error) {
	if blockNr == api.LatestBlockNumber || blockNr == api.PendingBlockNumber {
		return a.blockReader.GetLatestBlockNumber(ctx)
	}
	if blockNr == api.EarliestBlockNumber {
		return 0, nil
	}
	return blockNr.ToUint64()
}

// BlockNumber returns the current block number
func (a *BlockAPI) BlockNumber(ctx context.Context) (hexutil.Uint64, error) {
	number, err := a.blockReader.GetLatestBlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	return hexutil.Uint64(number), nil
}

// GetBlockByNumber returns a block by number
func (a *BlockAPI) GetBlockByNumber(ctx context.Context, blockNr string, fullTx bool) (*api.RPCBlock, error) {
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	number, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	block, err := a.blockReader.GetBlock(ctx, number)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block: %v", err)}
	}

	// For simplicity, using nil for total difficulty
	// In production, you'd calculate or store this
	return api.NewRPCBlock(block, fullTx, nil), nil
}

// GetBlockByHash returns a block by hash
func (a *BlockAPI) GetBlockByHash(ctx context.Context, blockHash common.Hash, fullTx bool) (*api.RPCBlock, error) {
	block, err := a.blockReader.GetBlockByHash(ctx, blockHash)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block: %v", err)}
	}

	return api.NewRPCBlock(block, fullTx, nil), nil
}

// GetBlockTransactionCountByNumber returns the number of transactions in a block by number
func (a *BlockAPI) GetBlockTransactionCountByNumber(ctx context.Context, blockNr string) (*hexutil.Uint64, error) {
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	number, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	count, err := a.blockReader.GetTransactionCount(ctx, number)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction count: %v", err)}
	}

	result := hexutil.Uint64(count)
	return &result, nil
}

// GetBlockTransactionCountByHash returns the number of transactions in a block by hash
func (a *BlockAPI) GetBlockTransactionCountByHash(ctx context.Context, blockHash common.Hash) (*hexutil.Uint64, error) {
	count, err := a.blockReader.GetTransactionCountByHash(ctx, blockHash)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction count: %v", err)}
	}

	result := hexutil.Uint64(count)
	return &result, nil
}

// GetUncleCountByBlockNumber returns the number of uncles in a block by number
// Always returns 0 for BSC/PoS chains
func (a *BlockAPI) GetUncleCountByBlockNumber(ctx context.Context, blockNr string) (hexutil.Uint64, error) {
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return 0, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	number, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return 0, err
	}

	// Verify block exists
	_, err = a.blockReader.GetHeader(ctx, number)
	if err == storage.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block: %v", err)}
	}

	return 0, nil
}

// GetUncleCountByBlockHash returns the number of uncles in a block by hash
// Always returns 0 for BSC/PoS chains
func (a *BlockAPI) GetUncleCountByBlockHash(ctx context.Context, blockHash common.Hash) (hexutil.Uint64, error) {
	// Verify block exists
	_, err := a.blockReader.GetBlockByHash(ctx, blockHash)
	if err == storage.ErrNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block: %v", err)}
	}

	return 0, nil
}
