package txpool

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sunvim/evm_rpc/pkg/api"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// TxPoolAPI provides transaction pool inspection methods
type TxPoolAPI struct {
	txPool *storage.TxPoolStorage
}

// NewTxPoolAPI creates a new TxPoolAPI
func NewTxPoolAPI(txPool *storage.TxPoolStorage) *TxPoolAPI {
	return &TxPoolAPI{
		txPool: txPool,
	}
}

// Status returns the number of pending and queued transactions
func (api *TxPoolAPI) Status(ctx context.Context) (map[string]hexutil.Uint, error) {
	status, err := api.txPool.GetPoolStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool status: %w", err)
	}

	return map[string]hexutil.Uint{
		"pending": hexutil.Uint(status["pending"]),
		"queued":  hexutil.Uint(status["queued"]),
	}, nil
}

// Content returns the transactions contained within the transaction pool
func (a *TxPoolAPI) Content(ctx context.Context) (map[string]map[string]map[string]*api.RPCTransaction, error) {
	content, err := a.txPool.GetPoolContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool content: %w", err)
	}

	// Convert to RPC format
	result := make(map[string]map[string]map[string]*api.RPCTransaction)
	
	// Process pending transactions
	result["pending"] = make(map[string]map[string]*api.RPCTransaction)
	for addr, txsByNonce := range content["pending"] {
		result["pending"][addr] = make(map[string]*api.RPCTransaction)
		for nonceStr, tx := range txsByNonce {
			result["pending"][addr][nonceStr] = api.NewRPCPendingTransaction(tx)
		}
	}

	// Process queued transactions
	result["queued"] = make(map[string]map[string]*api.RPCTransaction)
	for addr, txsByNonce := range content["queued"] {
		result["queued"][addr] = make(map[string]*api.RPCTransaction)
		for nonceStr, tx := range txsByNonce {
			result["queued"][addr][nonceStr] = api.NewRPCPendingTransaction(tx)
		}
	}

	return result, nil
}

// Inspect returns a summary of transactions in the pool
func (api *TxPoolAPI) Inspect(ctx context.Context) (map[string]map[string]map[string]string, error) {
	content, err := api.txPool.GetPoolContent(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool content: %w", err)
	}

	// Convert to inspect format (address -> nonce -> summary string)
	result := make(map[string]map[string]map[string]string)
	
	// Process pending transactions
	result["pending"] = make(map[string]map[string]string)
	for addr, txsByNonce := range content["pending"] {
		result["pending"][addr] = make(map[string]string)
		for nonceStr, tx := range txsByNonce {
			result["pending"][addr][nonceStr] = formatTxSummary(tx)
		}
	}

	// Process queued transactions
	result["queued"] = make(map[string]map[string]string)
	for addr, txsByNonce := range content["queued"] {
		result["queued"][addr] = make(map[string]string)
		for nonceStr, tx := range txsByNonce {
			result["queued"][addr][nonceStr] = formatTxSummary(tx)
		}
	}

	return result, nil
}

// formatTxSummary formats a transaction into a summary string
func formatTxSummary(tx *types.Transaction) string {
	to := "contract creation"
	if tx.To() != nil {
		to = tx.To().Hex()
	}
	
	gasPrice := tx.GasPrice()
	if gasPrice == nil {
		gasPrice = tx.GasFeeCap()
	}
	
	return fmt.Sprintf("%s: %s wei + %s gas × %s wei", 
		to,
		tx.Value().String(),
		strconv.FormatUint(tx.Gas(), 10),
		gasPrice.String())
}
