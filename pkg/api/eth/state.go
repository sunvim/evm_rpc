package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// StateAPI provides state-related RPC methods
type StateAPI struct {
	blockReader *storage.BlockReader
	stateReader *storage.StateReader
	chainID     uint64
}

// NewStateAPI creates a new StateAPI
func NewStateAPI(blockReader *storage.BlockReader, stateReader *storage.StateReader, chainID uint64) *StateAPI {
	return &StateAPI{
		blockReader: blockReader,
		stateReader: stateReader,
		chainID:     chainID,
	}
}

// resolveBlockNumber resolves a block number tag to actual block number string
func (a *StateAPI) resolveBlockNumber(ctx context.Context, blockNr api.BlockNumber) (string, error) {
	if blockNr == api.LatestBlockNumber {
		return "latest", nil
	}
	if blockNr == api.PendingBlockNumber {
		return "pending", nil
	}
	if blockNr == api.EarliestBlockNumber {
		return "0", nil
	}
	
	num, err := blockNr.ToUint64()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", num), nil
}

// GetBalance returns the balance of an account at a given block
func (a *StateAPI) GetBalance(ctx context.Context, address common.Address, blockNr string) (*hexutil.Big, error) {
	// Parse block number
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	blockNumStr, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	balance, err := a.stateReader.GetBalance(ctx, address, blockNumStr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get balance: %v", err)}
	}

	return (*hexutil.Big)(balance), nil
}

// GetCode returns the code of an account at a given block
func (a *StateAPI) GetCode(ctx context.Context, address common.Address, blockNr string) (hexutil.Bytes, error) {
	// Parse block number
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	blockNumStr, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	code, err := a.stateReader.GetCode(ctx, address, blockNumStr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get code: %v", err)}
	}

	return code, nil
}

// GetStorageAt returns the storage value at a given key for an account at a given block
func (a *StateAPI) GetStorageAt(ctx context.Context, address common.Address, key common.Hash, blockNr string) (hexutil.Bytes, error) {
	// Parse block number
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	blockNumStr, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	value, err := a.stateReader.GetStorageAt(ctx, address, key, blockNumStr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get storage: %v", err)}
	}

	// Ensure the result is 32 bytes
	result := make([]byte, 32)
	copy(result[32-len(value):], value)
	
	return result, nil
}

// GetTransactionCount returns the nonce of an account at a given block
func (a *StateAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNr string) (hexutil.Uint64, error) {
	// Parse block number
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return 0, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	blockNumStr, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return 0, err
	}

	nonce, err := a.stateReader.GetNonce(ctx, address, blockNumStr)
	if err != nil {
		return 0, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get nonce: %v", err)}
	}

	return hexutil.Uint64(nonce), nil
}
