package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// GasAPI provides gas-related RPC methods
type GasAPI struct {
	blockReader *storage.BlockReader
	chainID     uint64
}

// NewGasAPI creates a new GasAPI
func NewGasAPI(blockReader *storage.BlockReader, chainID uint64) *GasAPI {
	return &GasAPI{
		blockReader: blockReader,
		chainID:     chainID,
	}
}

// GasPrice returns the current gas price
// For now, returns a fixed value of 5 gwei
func (api *GasAPI) GasPrice(ctx context.Context) (*hexutil.Big, error) {
	// 5 gwei = 5000000000 wei
	gasPrice := big.NewInt(5000000000)
	return (*hexutil.Big)(gasPrice), nil
}

// MaxPriorityFeePerGas returns the current max priority fee per gas
// For now, returns a fixed value of 1 gwei
func (api *GasAPI) MaxPriorityFeePerGas(ctx context.Context) (*hexutil.Big, error) {
	// 1 gwei = 1000000000 wei
	priorityFee := big.NewInt(1000000000)
	return (*hexutil.Big)(priorityFee), nil
}

// FeeHistory returns the fee history
func (a *GasAPI) FeeHistory(ctx context.Context, blockCount hexutil.Uint64, lastBlock string, rewardPercentiles []float64) (*api.FeeHistoryResult, error) {
	// Parse last block
	bn, err := api.ParseBlockNumber(lastBlock)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	// Resolve to actual block number
	var endBlock uint64
	if bn == api.LatestBlockNumber || bn == api.PendingBlockNumber {
		endBlock, err = a.blockReader.GetLatestBlockNumber(ctx)
		if err != nil {
			return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get latest block: %v", err)}
		}
	} else if bn == api.EarliestBlockNumber {
		endBlock = 0
	} else {
		endBlock, err = bn.ToUint64()
		if err != nil {
			return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
		}
	}

	// Calculate start block
	count := uint64(blockCount)
	if count == 0 {
		count = 1
	}
	if count > 1024 {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: "block count too large (max 1024)"}
	}

	var startBlock uint64
	if endBlock >= count-1 {
		startBlock = endBlock - count + 1
	} else {
		startBlock = 0
		count = endBlock + 1
	}

	// Build result with mock data
	result := &api.FeeHistoryResult{
		OldestBlock:  (*hexutil.Big)(big.NewInt(int64(startBlock))),
		BaseFeePerGas: make([]*hexutil.Big, count+1),
		GasUsedRatio: make([]float64, count),
	}

	// Mock base fee (5 gwei for all blocks)
	baseFee := big.NewInt(5000000000)
	for i := range result.BaseFeePerGas {
		result.BaseFeePerGas[i] = (*hexutil.Big)(new(big.Int).Set(baseFee))
	}

	// Mock gas used ratio (50% for all blocks)
	for i := range result.GasUsedRatio {
		result.GasUsedRatio[i] = 0.5
	}

	// Add reward data if requested
	if len(rewardPercentiles) > 0 {
		result.Reward = make([][]*hexutil.Big, count)
		priorityFee := big.NewInt(1000000000) // 1 gwei
		
		for i := range result.Reward {
			result.Reward[i] = make([]*hexutil.Big, len(rewardPercentiles))
			for j := range rewardPercentiles {
				// Return same priority fee for all percentiles (mock)
				result.Reward[i][j] = (*hexutil.Big)(new(big.Int).Set(priorityFee))
			}
		}
	}

	return result, nil
}

// EstimateGas estimates the gas needed for a transaction
// This is a placeholder - full implementation would require EVM execution
func (api *GasAPI) EstimateGas(ctx context.Context, args api.CallArgs) (hexutil.Uint64, error) {
	// Simple estimation: 21000 for transfers, 50000 for contract calls
	if args.Data == nil || len(*args.Data) == 0 {
		return hexutil.Uint64(21000), nil
	}
	return hexutil.Uint64(50000), nil
}
