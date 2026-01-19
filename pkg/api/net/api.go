package net

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// NetAPI provides network-related RPC methods
type NetAPI struct {
	networkID uint64
}

// NewNetAPI creates a new NetAPI
func NewNetAPI(networkID uint64) *NetAPI {
	return &NetAPI{
		networkID: networkID,
	}
}

// Version returns the current network ID
func (api *NetAPI) Version(ctx context.Context) (string, error) {
	return fmt.Sprintf("%d", api.networkID), nil
}

// Listening returns true if the client is actively listening for network connections
func (api *NetAPI) Listening(ctx context.Context) (bool, error) {
	// Always return true for a running RPC server
	return true, nil
}

// PeerCount returns the number of connected peers
func (api *NetAPI) PeerCount(ctx context.Context) (hexutil.Uint64, error) {
	// For now, return 0 as this is a read-only RPC service
	// In a full node, this would return the actual peer count
	return hexutil.Uint64(0), nil
}
