# EVM RPC API Handlers

Production-grade JSON-RPC 2.0 API handlers for an EVM blockchain RPC service.

## Overview

This package implements all standard Ethereum JSON-RPC APIs organized by namespace:

- **eth**: Core Ethereum APIs (blocks, transactions, state, gas)
- **net**: Network information APIs
- **web3**: Web3 utility APIs
- **txpool**: Transaction pool inspection APIs

## Architecture

```
pkg/api/
├── types.go           # Common RPC types, error codes, and utilities
├── eth/
│   ├── block.go       # Block query APIs
│   ├── transaction.go # Transaction query APIs
│   ├── state.go       # State query APIs
│   ├── txpool.go      # Transaction submission (eth_sendRawTransaction)
│   └── gas.go         # Gas estimation and fee history
├── net/
│   └── api.go         # Network APIs
├── web3/
│   └── api.go         # Web3 utility APIs
└── txpool/
    └── api.go         # Transaction pool inspection

pkg/rpc/
└── backend.go         # API backend initialization
```

## Implemented APIs

### Eth Namespace (Block APIs)

- `eth_blockNumber` - Get latest block number
- `eth_getBlockByNumber` - Get block by number (with/without full transactions)
- `eth_getBlockByHash` - Get block by hash (with/without full transactions)
- `eth_getBlockTransactionCountByNumber` - Get transaction count in a block by number
- `eth_getBlockTransactionCountByHash` - Get transaction count in a block by hash
- `eth_getUncleCountByBlockNumber` - Get uncle count (always returns 0 for PoS chains)
- `eth_getUncleCountByBlockHash` - Get uncle count by hash (always returns 0)

### Eth Namespace (Transaction APIs)

- `eth_getTransactionByHash` - Get transaction by hash
- `eth_getTransactionByBlockHashAndIndex` - Get transaction by block hash and index
- `eth_getTransactionByBlockNumberAndIndex` - Get transaction by block number and index
- `eth_getTransactionReceipt` - Get transaction receipt
- `eth_getTransactionCount` - Get account nonce

### Eth Namespace (State APIs)

- `eth_getBalance` - Get account balance at a given block
- `eth_getCode` - Get contract code at a given block
- `eth_getStorageAt` - Get storage value at a key for an account
- `eth_getTransactionCount` - Get account transaction count (nonce)

### Eth Namespace (Transaction Pool APIs)

- `eth_sendRawTransaction` - Submit a raw signed transaction
  - Validates transaction signature
  - Verifies chain ID
  - Checks nonce (must be >= current nonce)
  - Verifies sufficient balance for value + gas
  - Validates minimum gas limit (21000)
- `eth_pendingTransactions` - Get all pending transactions

### Eth Namespace (Gas APIs)

- `eth_gasPrice` - Get current gas price (returns 5 gwei)
- `eth_maxPriorityFeePerGas` - Get max priority fee per gas (returns 1 gwei)
- `eth_feeHistory` - Get historical gas fee data
- `eth_estimateGas` - Estimate gas for a transaction (basic implementation)

### Net Namespace

- `net_version` - Get network ID
- `net_listening` - Check if node is listening (always returns true)
- `net_peerCount` - Get peer count (returns 0 for read-only service)

### Web3 Namespace

- `web3_clientVersion` - Get client version string
- `web3_sha3` - Calculate Keccak-256 hash of data

### Txpool Namespace

- `txpool_status` - Get transaction pool status (pending/queued counts)
- `txpool_content` - Get full transaction pool content
- `txpool_inspect` - Get transaction pool summary

## Block Number Tags

All APIs support standard Ethereum block number tags:

- `"latest"` - Latest mined block
- `"earliest"` - Genesis block (block 0)
- `"pending"` - Pending state (treated as latest)
- `"0x..."` - Hex-encoded block number

## Error Codes

Standard JSON-RPC 2.0 error codes:

- `-32700` - Parse error
- `-32600` - Invalid request
- `-32601` - Method not found
- `-32602` - Invalid params
- `-32603` - Internal error

Ethereum-specific error codes:

- `-32000` - Unknown block
- `-32001` - Invalid input
- `-32002` - Resource not found
- `-32003` - Resource unavailable
- `-32004` - Transaction rejected
- `-32005` - Method not supported
- `-32006` - Limit exceeded
- `-32007` - Version not supported

## Usage

```go
import (
    "github.com/sunvim/evm_rpc/pkg/config"
    "github.com/sunvim/evm_rpc/pkg/storage"
    "github.com/sunvim/evm_rpc/pkg/rpc"
)

// Initialize storage
cfg := &config.PikaConfig{
    Addr: "localhost:9221",
    // ... other config
}
pikaClient, err := storage.NewPikaClient(cfg)
if err != nil {
    log.Fatal(err)
}

// Create API backend
backend := rpc.NewAPIBackend(
    pikaClient,
    56,        // chainID (BSC mainnet)
    56,        // networkID
    "1.0.0",   // version
)

// Use APIs
ctx := context.Background()

// Get latest block number
blockNum, err := backend.BlockAPI.BlockNumber(ctx)

// Get block by number
block, err := backend.BlockAPI.GetBlockByNumber(ctx, "latest", true)

// Get account balance
balance, err := backend.StateAPI.GetBalance(ctx, address, "latest")

// Submit transaction
txHash, err := backend.TxPoolAPI.SendRawTransaction(ctx, signedTxBytes)
```

## Features

### Transaction Validation

`eth_sendRawTransaction` performs comprehensive validation:

1. **Signature Verification**: Validates transaction signature and recovers sender
2. **Chain ID Check**: Ensures transaction chain ID matches configured chain
3. **Nonce Validation**: Verifies nonce is >= current account nonce
4. **Balance Check**: Confirms account has sufficient balance for value + gas cost
5. **Gas Limit**: Validates minimum gas limit of 21000

### Block Tag Resolution

All state queries support flexible block number specifications:

- Block tags: `"latest"`, `"earliest"`, `"pending"`
- Hex numbers: `"0x1"`, `"0xa"`, `"0x3e8"`
- Automatic resolution to actual block numbers

### RPC Response Formats

All responses follow Ethereum JSON-RPC specification:

- Blocks: `RPCBlock` with optional full transaction objects
- Transactions: `RPCTransaction` with all EIP-1559 fields
- Receipts: `RPCReceipt` with logs, status, and effective gas price
- Errors: Standard `RPCError` with code and message

## Testing

```bash
# Build all API packages
go build ./pkg/api/...

# Build entire project
go build ./...

# Run tests (when available)
go test ./pkg/api/...
```

## Dependencies

- `github.com/ethereum/go-ethereum` - Core Ethereum types and utilities
- `github.com/redis/go-redis/v9` - Pika/Redis client (via storage layer)

## Notes

### Mock Data

Some APIs return mock data for simplicity:

- `eth_gasPrice`: Fixed 5 gwei
- `eth_maxPriorityFeePerGas`: Fixed 1 gwei  
- `eth_feeHistory`: Mock historical data with 50% gas usage ratio
- `eth_estimateGas`: Simple estimation (21000 for transfers, 50000 for contracts)

### Uncle Blocks

BSC and other PoS chains don't have uncle blocks:

- `eth_getUncleCountByBlockNumber`: Always returns 0
- `eth_getUncleCountByBlockHash`: Always returns 0

### Production Considerations

For production use, you should:

1. Implement dynamic gas price estimation based on recent blocks
2. Add EVM execution for accurate `eth_estimateGas`
3. Store and return total difficulty for blocks
4. Implement `eth_call` for contract calls
5. Add rate limiting per method
6. Implement request logging and metrics
7. Add comprehensive error handling
8. Implement caching for frequently accessed data

## Integration

These API handlers are designed to be integrated with a JSON-RPC server (HTTP/WebSocket). The server should:

1. Parse JSON-RPC requests
2. Route to appropriate API method
3. Handle authentication/authorization
4. Apply rate limiting
5. Return JSON-RPC responses
6. Log requests and errors

See the `pkg/server` package for JSON-RPC server implementation (to be created).
