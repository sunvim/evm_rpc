package rpc

import (
	"github.com/sunvim/evm_rpc/pkg/api/eth"
	"github.com/sunvim/evm_rpc/pkg/api/net"
	"github.com/sunvim/evm_rpc/pkg/api/txpool"
	"github.com/sunvim/evm_rpc/pkg/api/web3"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// APIBackend holds all API namespaces
type APIBackend struct {
	// Eth namespace
	BlockAPI       *eth.BlockAPI
	TransactionAPI *eth.TransactionAPI
	StateAPI       *eth.StateAPI
	TxPoolAPI      *eth.TxPoolAPI
	GasAPI         *eth.GasAPI

	// Net namespace
	NetAPI *net.NetAPI

	// Web3 namespace
	Web3API *web3.Web3API

	// Txpool namespace
	TxPoolInspectAPI *txpool.TxPoolAPI
}

// NewAPIBackend creates a new API backend with all namespaces
func NewAPIBackend(
	pikaClient *storage.PikaClient,
	chainID uint64,
	networkID uint64,
	version string,
) *APIBackend {
	// Create storage readers
	blockReader := storage.NewBlockReader(pikaClient)
	txReader := storage.NewTransactionReader(pikaClient)
	stateReader := storage.NewStateReader(pikaClient)
	txPool := storage.NewTxPoolStorage(pikaClient)

	return &APIBackend{
		// Eth namespace
		BlockAPI:       eth.NewBlockAPI(blockReader, chainID),
		TransactionAPI: eth.NewTransactionAPI(blockReader, txReader, chainID),
		StateAPI:       eth.NewStateAPI(blockReader, stateReader, chainID),
		TxPoolAPI:      eth.NewTxPoolAPI(blockReader, stateReader, txPool, chainID),
		GasAPI:         eth.NewGasAPI(blockReader, chainID),

		// Net namespace
		NetAPI: net.NewNetAPI(networkID),

		// Web3 namespace
		Web3API: web3.NewWeb3API(version),

		// Txpool namespace
		TxPoolInspectAPI: txpool.NewTxPoolAPI(txPool),
	}
}
