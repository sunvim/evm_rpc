package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// TransactionAPI provides transaction-related RPC methods
type TransactionAPI struct {
	blockReader *storage.BlockReader
	txReader    *storage.TransactionReader
	chainID     uint64
}

// NewTransactionAPI creates a new TransactionAPI
func NewTransactionAPI(blockReader *storage.BlockReader, txReader *storage.TransactionReader, chainID uint64) *TransactionAPI {
	return &TransactionAPI{
		blockReader: blockReader,
		txReader:    txReader,
		chainID:     chainID,
	}
}

// resolveBlockNumber resolves a block number tag to actual block number
func (a *TransactionAPI) resolveBlockNumber(ctx context.Context, blockNr api.BlockNumber) (uint64, error) {
	if blockNr == api.LatestBlockNumber || blockNr == api.PendingBlockNumber {
		return a.blockReader.GetLatestBlockNumber(ctx)
	}
	if blockNr == api.EarliestBlockNumber {
		return 0, nil
	}
	return blockNr.ToUint64()
}

// GetTransactionByHash returns a transaction by hash
func (a *TransactionAPI) GetTransactionByHash(ctx context.Context, txHash common.Hash) (*api.RPCTransaction, error) {
	// Get transaction
	tx, err := a.txReader.GetTransaction(ctx, txHash)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction: %v", err)}
	}

	// Get lookup information
	lookup, err := a.txReader.GetTransactionLookup(ctx, txHash)
	if err == storage.ErrNotFound {
		// Transaction exists but not yet included in a block
		return api.NewRPCPendingTransaction(tx), nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction lookup: %v", err)}
	}

	blockHash := common.HexToHash(lookup.BlockHash)
	return api.NewRPCTransaction(tx, blockHash, lookup.BlockNumber, lookup.Index), nil
}

// GetTransactionByBlockHashAndIndex returns a transaction by block hash and index
func (a *TransactionAPI) GetTransactionByBlockHashAndIndex(ctx context.Context, blockHash common.Hash, index hexutil.Uint64) (*api.RPCTransaction, error) {
	tx, err := a.txReader.GetTransactionByBlockHashAndIndex(ctx, blockHash, uint64(index))
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction: %v", err)}
	}

	// Get block number
	blockNumber, err := a.blockReader.GetBlockNumberByHash(ctx, blockHash)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block number: %v", err)}
	}

	return api.NewRPCTransaction(tx, blockHash, blockNumber, uint64(index)), nil
}

// GetTransactionByBlockNumberAndIndex returns a transaction by block number and index
func (a *TransactionAPI) GetTransactionByBlockNumberAndIndex(ctx context.Context, blockNr string, index hexutil.Uint64) (*api.RPCTransaction, error) {
	bn, err := api.ParseBlockNumber(blockNr)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInvalidParams, Message: fmt.Sprintf("invalid block number: %v", err)}
	}

	number, err := a.resolveBlockNumber(ctx, bn)
	if err != nil {
		return nil, err
	}

	tx, err := a.txReader.GetTransactionByBlockNumberAndIndex(ctx, number, uint64(index))
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction: %v", err)}
	}

	// Get block hash
	header, err := a.blockReader.GetHeader(ctx, number)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get block header: %v", err)}
	}

	return api.NewRPCTransaction(tx, header.Hash(), number, uint64(index)), nil
}

// GetTransactionReceipt returns a transaction receipt by hash
func (a *TransactionAPI) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*api.RPCReceipt, error) {
	// Get receipt and lookup
	receipt, lookup, err := a.txReader.GetReceipt(ctx, txHash)
	if err == storage.ErrNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get receipt: %v", err)}
	}

	// Get transaction
	tx, err := a.txReader.GetTransaction(ctx, txHash)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get transaction: %v", err)}
	}

	blockHash := common.HexToHash(lookup.BlockHash)
	return api.NewRPCReceipt(receipt, tx, blockHash, lookup.BlockNumber, lookup.Index), nil
}

// GetTransactionCount returns the nonce of an account at a given block
func (a *TransactionAPI) GetTransactionCount(ctx context.Context, address common.Address, blockNr string) (hexutil.Uint64, error) {
	// This is handled by StateAPI, but included here for reference
	// In practice, this would call the state reader
	return 0, &api.RPCError{Code: api.ErrCodeMethodNotSupported, Message: "use StateAPI.GetTransactionCount"}
}
