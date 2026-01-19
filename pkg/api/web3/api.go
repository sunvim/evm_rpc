package web3

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

// Web3API provides web3-related RPC methods
type Web3API struct {
	version string
}

// NewWeb3API creates a new Web3API
func NewWeb3API(version string) *Web3API {
	if version == "" {
		version = "1.0.0"
	}
	return &Web3API{
		version: version,
	}
}

// ClientVersion returns the current client version
func (api *Web3API) ClientVersion(ctx context.Context) (string, error) {
	return fmt.Sprintf("evm-rpc/%s", api.version), nil
}

// Sha3 returns the Keccak-256 hash of the given data
func (api *Web3API) Sha3(ctx context.Context, input hexutil.Bytes) (hexutil.Bytes, error) {
	return crypto.Keccak256(input), nil
}
