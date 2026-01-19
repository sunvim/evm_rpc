package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// StateReader reads state data from Pika
type StateReader struct {
	client *PikaClient
}

// NewStateReader creates a new state reader
func NewStateReader(client *PikaClient) *StateReader {
	return &StateReader{client: client}
}

// AccountState represents account state
type AccountState struct {
	Nonce    uint64   `json:"nonce"`
	Balance  *big.Int `json:"balance"`
	CodeHash string   `json:"codeHash"`
}

// GetBalance returns account balance at block number
func (r *StateReader) GetBalance(ctx context.Context, address common.Address, blockNumber string) (*big.Int, error) {
	var key string
	if blockNumber == "latest" || blockNumber == "pending" {
		key = fmt.Sprintf("st:latest:acc:%s", address.Hex())
	} else {
		// Parse block number
		key = fmt.Sprintf("st:%s:acc:%s", blockNumber, address.Hex())
	}

	data, err := r.client.Get(ctx, key)
	if err == ErrNotFound {
		// Account doesn't exist, return 0
		return big.NewInt(0), nil
	}
	if err != nil {
		return nil, err
	}

	var state AccountState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to decode account state: %w", err)
	}

	if state.Balance == nil {
		return big.NewInt(0), nil
	}

	return state.Balance, nil
}

// GetNonce returns account nonce at block number
func (r *StateReader) GetNonce(ctx context.Context, address common.Address, blockNumber string) (uint64, error) {
	var key string
	if blockNumber == "latest" || blockNumber == "pending" {
		key = fmt.Sprintf("st:latest:acc:%s", address.Hex())
	} else {
		key = fmt.Sprintf("st:%s:acc:%s", blockNumber, address.Hex())
	}

	data, err := r.client.Get(ctx, key)
	if err == ErrNotFound {
		// Account doesn't exist, return 0
		return 0, nil
	}
	if err != nil {
		return 0, err
	}

	var state AccountState
	if err := json.Unmarshal(data, &state); err != nil {
		return 0, fmt.Errorf("failed to decode account state: %w", err)
	}

	return state.Nonce, nil
}

// GetCode returns contract code
func (r *StateReader) GetCode(ctx context.Context, address common.Address, blockNumber string) ([]byte, error) {
	// First get code hash from account state
	var accKey string
	if blockNumber == "latest" || blockNumber == "pending" {
		accKey = fmt.Sprintf("st:latest:acc:%s", address.Hex())
	} else {
		accKey = fmt.Sprintf("st:%s:acc:%s", blockNumber, address.Hex())
	}

	accData, err := r.client.Get(ctx, accKey)
	if err == ErrNotFound {
		// No code
		return []byte{}, nil
	}
	if err != nil {
		return nil, err
	}

	var state AccountState
	if err := json.Unmarshal(accData, &state); err != nil {
		return nil, fmt.Errorf("failed to decode account state: %w", err)
	}

	emptyHash := common.Hash{}.Hex()
	if state.CodeHash == "" || state.CodeHash == emptyHash {
		return []byte{}, nil
	}

	// Get code by hash
	codeKey := fmt.Sprintf("st:code:%s", state.CodeHash)
	code, err := r.client.Get(ctx, codeKey)
	if err == ErrNotFound {
		return []byte{}, nil
	}
	if err != nil {
		return nil, err
	}

	return code, nil
}

// GetStorageAt returns storage value at key
func (r *StateReader) GetStorageAt(ctx context.Context, address common.Address, key common.Hash, blockNumber string) ([]byte, error) {
	var storageKey string
	if blockNumber == "latest" || blockNumber == "pending" {
		storageKey = fmt.Sprintf("st:latest:stor:%s:%s", address.Hex(), key.Hex())
	} else {
		storageKey = fmt.Sprintf("st:%s:stor:%s:%s", blockNumber, address.Hex(), key.Hex())
	}

	value, err := r.client.Get(ctx, storageKey)
	if err == ErrNotFound {
		// Storage slot is empty
		return common.Hash{}.Bytes(), nil
	}
	if err != nil {
		return nil, err
	}

	return value, nil
}

// GetAccountState returns full account state
func (r *StateReader) GetAccountState(ctx context.Context, address common.Address, blockNumber string) (*AccountState, error) {
	var key string
	if blockNumber == "latest" || blockNumber == "pending" {
		key = fmt.Sprintf("st:latest:acc:%s", address.Hex())
	} else {
		key = fmt.Sprintf("st:%s:acc:%s", blockNumber, address.Hex())
	}

	data, err := r.client.Get(ctx, key)
	if err == ErrNotFound {
		// Account doesn't exist
		return &AccountState{
			Nonce:    0,
			Balance:  big.NewInt(0),
			CodeHash: "",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var state AccountState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to decode account state: %w", err)
	}

	return &state, nil
}
