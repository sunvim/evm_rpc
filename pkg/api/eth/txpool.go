package eth

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// TxPoolAPI provides transaction pool related RPC methods
type TxPoolAPI struct {
	blockReader *storage.BlockReader
	stateReader *storage.StateReader
	txPool      *storage.TxPoolStorage
	chainID     uint64
}

// NewTxPoolAPI creates a new TxPoolAPI
func NewTxPoolAPI(blockReader *storage.BlockReader, stateReader *storage.StateReader, txPool *storage.TxPoolStorage, chainID uint64) *TxPoolAPI {
	return &TxPoolAPI{
		blockReader: blockReader,
		stateReader: stateReader,
		txPool:      txPool,
		chainID:     chainID,
	}
}

// SendRawTransaction submits a raw transaction
func (a *TxPoolAPI) SendRawTransaction(ctx context.Context, input hexutil.Bytes) (common.Hash, error) {
	// Decode transaction
	tx := new(types.Transaction)
	if err := rlp.DecodeBytes(input, tx); err != nil {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInvalidInput, Message: fmt.Sprintf("invalid transaction: %v", err)}
	}

	// Validate transaction signature
	signer := types.LatestSignerForChainID(tx.ChainId())
	from, err := types.Sender(signer, tx)
	if err != nil {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInvalidInput, Message: fmt.Sprintf("invalid signature: %v", err)}
	}

	// Verify chain ID
	if tx.ChainId() != nil && tx.ChainId().Uint64() != a.chainID {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInvalidInput, Message: 
			fmt.Sprintf("invalid chain id: got %d, expected %d", tx.ChainId().Uint64(), a.chainID)}
	}

	// Get current account nonce
	currentNonce, err := a.stateReader.GetNonce(ctx, from, "latest")
	if err != nil {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get nonce: %v", err)}
	}

	// Check nonce (must be >= current nonce)
	if tx.Nonce() < currentNonce {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeTransactionReject, Message: 
			fmt.Sprintf("nonce too low: got %d, expected >= %d", tx.Nonce(), currentNonce)}
	}

	// Get account balance
	balance, err := a.stateReader.GetBalance(ctx, from, "latest")
	if err != nil {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get balance: %v", err)}
	}

	// Calculate total cost (value + gas)
	gasPrice := tx.GasPrice()
	if gasPrice == nil {
		gasPrice = tx.GasFeeCap()
	}
	if gasPrice == nil {
		gasPrice = big.NewInt(0)
	}

	gasCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(tx.Gas())))
	totalCost := new(big.Int).Add(tx.Value(), gasCost)

	// Check balance
	if balance.Cmp(totalCost) < 0 {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeTransactionReject, Message: 
			fmt.Sprintf("insufficient funds: balance=%s, required=%s", balance.String(), totalCost.String())}
	}

	// Validate gas limit
	if tx.Gas() < 21000 {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInvalidInput, Message: 
			fmt.Sprintf("gas limit too low: got %d, minimum 21000", tx.Gas())}
	}

	// Add to transaction pool
	if err := a.txPool.AddPendingTx(ctx, tx, "rpc"); err != nil {
		return common.Hash{}, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to add transaction: %v", err)}
	}

	return tx.Hash(), nil
}

// PendingTransactions returns all pending transactions
func (a *TxPoolAPI) PendingTransactions(ctx context.Context) ([]*api.RPCTransaction, error) {
	txs, err := a.txPool.GetPendingTransactions(ctx)
	if err != nil {
		return nil, &api.RPCError{Code: api.ErrCodeInternal, Message: fmt.Sprintf("failed to get pending transactions: %v", err)}
	}

	result := make([]*api.RPCTransaction, len(txs))
	for i, tx := range txs {
		result[i] = api.NewRPCPendingTransaction(tx)
	}

	return result, nil
}
