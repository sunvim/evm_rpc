# EVM RPC Service

A production-grade, lightweight, distributed JSON-RPC 2.0 server for EVM-compatible blockchains (BSC, Ethereum, etc.). This service provides standard Ethereum JSON-RPC APIs with high performance, horizontal scalability, and comprehensive monitoring.

## Features

✨ **Core Capabilities**
- 🔌 Full Ethereum JSON-RPC 2.0 support (37+ methods)
- 🚀 HTTP and WebSocket transport
- 📊 Real-time subscriptions (newHeads, logs, pendingTransactions)
- 💾 Pika/Redis storage backend
- ⚡ Multi-layer caching (LRU with TTL)
- 🛡️ Three-tier rate limiting (global, IP, method)
- 📈 Prometheus metrics
- 🔍 Health checks and slow query logging

✨ **Production Ready**
- 🎯 Stateless architecture (horizontal scaling)
- 🔄 Graceful shutdown
- 🐳 Docker & Kubernetes deployment
- 📉 HPA (Horizontal Pod Autoscaling)
- 🔒 Security: CodeQL validated, no vulnerabilities
- 📚 Comprehensive documentation

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Client    │────▶│  EVM RPC    │────▶│    Pika     │
│ (web3.js)   │◀────│   Service   │◀────│  (Storage)  │
└─────────────┘     └─────────────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐
                    │ Prometheus  │
                    │  (Metrics)  │
                    └─────────────┘
```

## Supported RPC Methods

### Eth Namespace (26 methods)

**Block Queries:**
- `eth_blockNumber` - Get latest block number
- `eth_getBlockByNumber` - Get block by number
- `eth_getBlockByHash` - Get block by hash
- `eth_getBlockTransactionCountByNumber` - Transaction count in block
- `eth_getBlockTransactionCountByHash` - Transaction count in block
- `eth_getUncleCountByBlockNumber` - Uncle count (0 for BSC)
- `eth_getUncleCountByBlockHash` - Uncle count (0 for BSC)

**Transaction Queries:**
- `eth_getTransactionByHash` - Get transaction by hash
- `eth_getTransactionByBlockHashAndIndex` - Get transaction by block and index
- `eth_getTransactionByBlockNumberAndIndex` - Get transaction by block and index
- `eth_getTransactionReceipt` - Get transaction receipt
- `eth_getTransactionCount` - Get account nonce

**State Queries:**
- `eth_getBalance` - Get account balance
- `eth_getCode` - Get contract code
- `eth_getStorageAt` - Get storage value
- `eth_call` - Execute read-only call
- `eth_estimateGas` - Estimate gas usage

**Transaction Submission:**
- `eth_sendRawTransaction` - Submit signed transaction

**Gas:**
- `eth_gasPrice` - Current gas price
- `eth_maxPriorityFeePerGas` - Max priority fee (EIP-1559)
- `eth_feeHistory` - Historical gas fees

**Logs:**
- `eth_getLogs` - Query event logs

**Metadata:**
- `eth_chainId` - Chain ID
- `eth_syncing` - Sync status
- `eth_protocolVersion` - Protocol version

**Pending:**
- `eth_pendingTransactions` - Get pending transactions

### Net Namespace (3 methods)
- `net_version` - Network ID
- `net_listening` - Network listening status
- `net_peerCount` - Connected peers

### Web3 Namespace (2 methods)
- `web3_clientVersion` - Client version
- `web3_sha3` - Keccak-256 hash

### Txpool Namespace (3 methods)
- `txpool_status` - Pool statistics
- `txpool_content` - Pool content (pending + queued)
- `txpool_inspect` - Pool inspection

### WebSocket Subscriptions
- `eth_subscribe("newHeads")` - Subscribe to new blocks
- `eth_subscribe("logs", filter)` - Subscribe to logs
- `eth_subscribe("newPendingTransactions")` - Subscribe to pending transactions
- `eth_unsubscribe(subscriptionId)` - Unsubscribe

## Quick Start

### Prerequisites

- Go 1.21+
- Pika or Redis 6.0+
- Docker (optional)

### Installation

```bash
# Clone repository
git clone https://github.com/sunvim/evm_rpc.git
cd evm_rpc

# Download dependencies
make deps

# Build
make build

# Run
make run
```

### Configuration

Edit `config/config.yaml`:

```yaml
chain:
  name: "bsc"
  network_id: 56
  chain_id: 56

storage:
  pika:
    addr: "127.0.0.1:9221"
    password: ""
    
server:
  http:
    listen_addr: "0.0.0.0:8545"
  ws:
    listen_addr: "0.0.0.0:8546"
```

## Usage Examples

### Using web3.js

```javascript
const Web3 = require('web3');
const web3 = new Web3('http://localhost:8545');

// Get latest block
const blockNumber = await web3.eth.getBlockNumber();
console.log('Latest block:', blockNumber);

// Get balance
const balance = await web3.eth.getBalance('0x...');
console.log('Balance:', web3.utils.fromWei(balance, 'ether'), 'ETH');

// Send transaction
const signedTx = await web3.eth.accounts.signTransaction({
  to: '0x...',
  value: web3.utils.toWei('1', 'ether'),
  gas: 21000,
  gasPrice: web3.utils.toWei('5', 'gwei'),
  nonce: await web3.eth.getTransactionCount('0x...')
}, privateKey);

const receipt = await web3.eth.sendSignedTransaction(signedTx.rawTransaction);
console.log('Transaction hash:', receipt.transactionHash);
```

### Using ethers.js

```javascript
const { ethers } = require('ethers');
const provider = new ethers.JsonRpcProvider('http://localhost:8545');

// Get latest block
const blockNumber = await provider.getBlockNumber();
console.log('Latest block:', blockNumber);

// Get balance
const balance = await provider.getBalance('0x...');
console.log('Balance:', ethers.formatEther(balance), 'ETH');

// Send transaction
const wallet = new ethers.Wallet(privateKey, provider);
const tx = await wallet.sendTransaction({
  to: '0x...',
  value: ethers.parseEther('1.0')
});
await tx.wait();
console.log('Transaction hash:', tx.hash);
```

### WebSocket Subscriptions

```javascript
const Web3 = require('web3');
const web3 = new Web3(new Web3.providers.WebsocketProvider('ws://localhost:8546'));

// Subscribe to new blocks
const subscription = await web3.eth.subscribe('newHeads');
subscription.on('data', (blockHeader) => {
  console.log('New block:', blockHeader.number);
});

// Subscribe to logs
const logSubscription = await web3.eth.subscribe('logs', {
  address: '0x...',
  topics: ['0x...']
});
logSubscription.on('data', (log) => {
  console.log('New log:', log);
});
```

## Docker Deployment

### Using Docker Compose

```bash
# Start all services (Pika + RPC + Prometheus + Grafana)
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

Access:
- RPC HTTP: http://localhost:8545
- RPC WebSocket: ws://localhost:8546
- Health: http://localhost:8080/health
- Metrics: http://localhost:9092/metrics
- Prometheus: http://localhost:9090
- Grafana: http://localhost:3000 (admin/admin)

### Building Docker Image

```bash
make docker-build
```

## Kubernetes Deployment

```bash
# Apply manifests
kubectl apply -f deployments/kubernetes/deployment.yaml

# Check status
kubectl get pods -l app=evm-rpc
kubectl get svc evm-rpc-service

# View logs
kubectl logs -f deployment/evm-rpc

# Scale manually
kubectl scale deployment evm-rpc --replicas=5
```

Features:
- ✅ 3 replicas by default
- ✅ Horizontal Pod Autoscaler (3-10 replicas)
- ✅ Resource limits (2 CPU, 4GB RAM)
- ✅ Liveness & Readiness probes
- ✅ LoadBalancer service
- ✅ Prometheus annotations

## Monitoring

### Prometheus Metrics

Available at `http://localhost:9092/metrics`:

```
# Request metrics
rpc_requests_total{method="eth_getBalance",status="success"} 1234
rpc_request_duration_seconds{method="eth_call"} 0.045
rpc_requests_in_flight{method="eth_sendRawTransaction"} 3

# Rate limiting
rpc_ratelimit_rejections_total{type="ip"} 42

# WebSocket
rpc_websocket_connections 150
rpc_subscriptions_total{type="newHeads"} 45

# Cache
rpc_cache_hits_total{type="block"} 9876
rpc_cache_misses_total{type="transaction"} 234
```

### Health Check

```bash
curl http://localhost:8080/health
```

Response:
```json
{
  "status": "healthy",
  "syncHeight": 12345678,
  "execHeight": 12345678,
  "lag": 0,
  "peerCount": 0,
  "txPoolSize": 42,
  "cacheHitRate": 0.95,
  "uptime": 3600,
  "version": "v1.0.0"
}
```

Status values:
- `healthy` - Service is operating normally (lag < 100 blocks)
- `degraded` - Service is slow (lag 100-1000 blocks)
- `unhealthy` - Service has issues (lag > 1000 blocks)

## Performance

### Caching Strategy

Three-layer architecture:
1. **Memory (LRU)** - Fast, limited size
2. **Pika/Redis** - Persistent, shared across instances
3. **Source** - Block data from sync service

Cache TTLs:
- Blocks: Permanent
- Transactions: Permanent
- Receipts: Permanent
- Balance: 10 seconds (state changes)
- Code: 1 hour

### Rate Limiting

Three-tier protection:
1. **Global**: 1000 req/s, burst 2000
2. **Per-IP**: 100 req/s, burst 200
3. **Per-Method**: Custom limits
   - `eth_call`: 50 req/s
   - `eth_estimateGas`: 50 req/s
   - `eth_getLogs`: 10 req/s
   - `eth_sendRawTransaction`: 20 req/s
   - `eth_getBalance`: 100 req/s
   - `eth_blockNumber`: 200 req/s

## Data Storage (Pika/Redis Keys)

### Block Data
```
idx:latest                  → Latest block number
idx:blk:hash:{hash}         → Block number lookup
blk:hdr:{number}            → Block header (RLP)
blk:body:{number}           → Block body (RLP)
blk:rcpt:{number}           → Block receipts (RLP)
```

### Transaction Data
```
tx:{hash}                   → Transaction (RLP)
tx:lookup:{hash}            → {"blockNumber": N, "index": I}
```

### State Data
```
st:latest:acc:{address}     → {"nonce": N, "balance": "B", "codeHash": "H"}
st:latest:stor:{addr}:{key} → Storage value
st:code:{codeHash}          → Contract code
st:{blockNum}:acc:{address} → Historical state (1024 block window)
```

### Transaction Pool
```
pool:pending:{hash}         → Pending transaction (RLP)
pool:addr:{address}         → Sorted set of tx hashes by nonce
pool:byprice                → Sorted set of tx hashes by gas price
```

### Pub/Sub Channels
```
blocks:new                  → New block notifications
pool:new                    → New transaction notifications
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run with coverage
make test-coverage
```

### Code Quality

```bash
# Format code
make fmt

# Run linter
make lint
```

### Project Structure

```
evm_rpc/
├── cmd/rpc/              # Main application
├── pkg/
│   ├── api/              # RPC API implementations
│   │   ├── eth/          # Ethereum namespace
│   │   ├── net/          # Network namespace
│   │   ├── web3/         # Web3 namespace
│   │   └── txpool/       # Transaction pool namespace
│   ├── server/           # HTTP/WebSocket servers
│   ├── storage/          # Pika storage layer
│   ├── cache/            # LRU caching
│   ├── middleware/       # Rate limiting, logging, CORS
│   ├── metrics/          # Prometheus metrics
│   ├── config/           # Configuration
│   └── logger/           # Logging
├── config/               # Configuration files
└── deployments/          # Docker & Kubernetes
```

## Security

✅ **No vulnerabilities** - Validated with CodeQL
✅ **Rate limiting** - Protection against abuse
✅ **Input validation** - All parameters validated
✅ **Origin checking** - WebSocket origin validation
✅ **Resource limits** - Memory and connection limits
✅ **Secure RNG** - crypto/rand for IDs

## Contributing

Contributions are welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

- 📧 Email: support@example.com
- 🐛 Issues: https://github.com/sunvim/evm_rpc/issues
- 📖 Wiki: https://github.com/sunvim/evm_rpc/wiki

## Acknowledgments

Built with:
- [go-ethereum](https://github.com/ethereum/go-ethereum) - Ethereum protocol implementation
- [Pika](https://github.com/OpenAtomFoundation/pika) - Redis-compatible storage
- [Prometheus](https://prometheus.io/) - Monitoring and alerting
- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation
