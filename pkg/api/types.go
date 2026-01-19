package api

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Standard JSON-RPC 2.0 error codes
const (
	ErrCodeParse          = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
)

// Ethereum-specific error codes
const (
	ErrCodeUnknownBlock       = -32000
	ErrCodeInvalidInput       = -32001
	ErrCodeResourceNotFound   = -32002
	ErrCodeResourceUnavail    = -32003
	ErrCodeTransactionReject  = -32004
	ErrCodeMethodNotSupported = -32005
	ErrCodeLimitExceeded      = -32006
	ErrCodeVersionNotSupport  = -32007
)

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *RPCError) Error() string {
	return fmt.Sprintf("rpc error: code=%d, message=%s", e.Code, e.Message)
}

// NewRPCError creates a new RPC error
func NewRPCError(code int, message string) *RPCError {
	return &RPCError{Code: code, Message: message}
}

// Common RPC errors
var (
	ErrInvalidParams       = NewRPCError(ErrCodeInvalidParams, "invalid params")
	ErrInternal            = NewRPCError(ErrCodeInternal, "internal error")
	ErrBlockNotFound       = NewRPCError(ErrCodeUnknownBlock, "block not found")
	ErrTransactionNotFound = NewRPCError(ErrCodeResourceNotFound, "transaction not found")
	ErrInvalidTransaction  = NewRPCError(ErrCodeInvalidInput, "invalid transaction")
)

// BlockNumber represents a block number parameter
type BlockNumber int64

const (
	LatestBlockNumber  = BlockNumber(-1)
	EarliestBlockNumber = BlockNumber(0)
	PendingBlockNumber = BlockNumber(-2)
)

// BlockNumberOrHash contains either a block number or a block hash
type BlockNumberOrHash struct {
	BlockNumber *BlockNumber  `json:"blockNumber,omitempty"`
	BlockHash   *common.Hash  `json:"blockHash,omitempty"`
	RequireCanonical bool     `json:"requireCanonical,omitempty"`
}

// ParseBlockNumber parses a block number string
func ParseBlockNumber(input string) (BlockNumber, error) {
	input = strings.TrimSpace(strings.ToLower(input))
	
	switch input {
	case "latest":
		return LatestBlockNumber, nil
	case "earliest":
		return EarliestBlockNumber, nil
	case "pending":
		return PendingBlockNumber, nil
	default:
		// Try to parse as hex number
		if !strings.HasPrefix(input, "0x") {
			return 0, fmt.Errorf("invalid block number: %s", input)
		}
		num, err := strconv.ParseUint(input[2:], 16, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hex block number: %w", err)
		}
		return BlockNumber(num), nil
	}
}

// ToUint64 converts BlockNumber to uint64
func (bn BlockNumber) ToUint64() (uint64, error) {
	if bn < 0 {
		return 0, fmt.Errorf("cannot convert %d to uint64", bn)
	}
	return uint64(bn), nil
}

// RPCBlock represents a block in RPC format
type RPCBlock struct {
	Number           *hexutil.Big      `json:"number"`
	Hash             *common.Hash      `json:"hash"`
	ParentHash       common.Hash       `json:"parentHash"`
	Nonce            *types.BlockNonce `json:"nonce"`
	Sha3Uncles       common.Hash       `json:"sha3Uncles"`
	LogsBloom        types.Bloom       `json:"logsBloom"`
	TransactionsRoot common.Hash       `json:"transactionsRoot"`
	StateRoot        common.Hash       `json:"stateRoot"`
	ReceiptsRoot     common.Hash       `json:"receiptsRoot"`
	Miner            common.Address    `json:"miner"`
	Difficulty       *hexutil.Big      `json:"difficulty"`
	TotalDifficulty  *hexutil.Big      `json:"totalDifficulty"`
	ExtraData        hexutil.Bytes     `json:"extraData"`
	Size             hexutil.Uint64    `json:"size"`
	GasLimit         hexutil.Uint64    `json:"gasLimit"`
	GasUsed          hexutil.Uint64    `json:"gasUsed"`
	Timestamp        hexutil.Uint64    `json:"timestamp"`
	Transactions     interface{}       `json:"transactions"`
	Uncles           []common.Hash     `json:"uncles"`
	MixHash          common.Hash       `json:"mixHash"`
	BaseFeePerGas    *hexutil.Big      `json:"baseFeePerGas,omitempty"`
}

// NewRPCBlock creates an RPCBlock from a types.Block
func NewRPCBlock(block *types.Block, fullTx bool, td *big.Int) *RPCBlock {
	head := block.Header()
	hash := head.Hash()
	
	rpcBlock := &RPCBlock{
		Number:           (*hexutil.Big)(head.Number),
		Hash:             &hash,
		ParentHash:       head.ParentHash,
		Nonce:            &head.Nonce,
		Sha3Uncles:       head.UncleHash,
		LogsBloom:        head.Bloom,
		TransactionsRoot: head.TxHash,
		StateRoot:        head.Root,
		ReceiptsRoot:     head.ReceiptHash,
		Miner:            head.Coinbase,
		Difficulty:       (*hexutil.Big)(head.Difficulty),
		ExtraData:        head.Extra,
		Size:             hexutil.Uint64(block.Size()),
		GasLimit:         hexutil.Uint64(head.GasLimit),
		GasUsed:          hexutil.Uint64(head.GasUsed),
		Timestamp:        hexutil.Uint64(head.Time),
		Uncles:           []common.Hash{},
		MixHash:          head.MixDigest,
	}

	if td != nil {
		rpcBlock.TotalDifficulty = (*hexutil.Big)(td)
	}

	if head.BaseFee != nil {
		rpcBlock.BaseFeePerGas = (*hexutil.Big)(head.BaseFee)
	}

	if fullTx {
		txs := make([]*RPCTransaction, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			txs[i] = NewRPCTransaction(tx, block.Hash(), block.NumberU64(), uint64(i))
		}
		rpcBlock.Transactions = txs
	} else {
		hashes := make([]common.Hash, len(block.Transactions()))
		for i, tx := range block.Transactions() {
			hashes[i] = tx.Hash()
		}
		rpcBlock.Transactions = hashes
	}

	return rpcBlock
}

// RPCTransaction represents a transaction in RPC format
type RPCTransaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	GasFeeCap        *hexutil.Big    `json:"maxFeePerGas,omitempty"`
	GasTipCap        *hexutil.Big    `json:"maxPriorityFeePerGas,omitempty"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	Type             hexutil.Uint64  `json:"type"`
	ChainID          *hexutil.Big    `json:"chainId,omitempty"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

// NewRPCTransaction creates an RPCTransaction from a types.Transaction
func NewRPCTransaction(tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCTransaction {
	v, r, s := tx.RawSignatureValues()
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

	result := &RPCTransaction{
		Type:     hexutil.Uint64(tx.Type()),
		From:     from,
		Gas:      hexutil.Uint64(tx.Gas()),
		GasPrice: (*hexutil.Big)(tx.GasPrice()),
		Hash:     tx.Hash(),
		Input:    tx.Data(),
		Nonce:    hexutil.Uint64(tx.Nonce()),
		To:       tx.To(),
		Value:    (*hexutil.Big)(tx.Value()),
		V:        (*hexutil.Big)(v),
		R:        (*hexutil.Big)(r),
		S:        (*hexutil.Big)(s),
	}

	if blockHash != (common.Hash{}) {
		result.BlockHash = &blockHash
		result.BlockNumber = (*hexutil.Big)(big.NewInt(int64(blockNumber)))
		idx := hexutil.Uint64(index)
		result.TransactionIndex = &idx
	}

	if tx.Type() == types.DynamicFeeTxType {
		result.GasFeeCap = (*hexutil.Big)(tx.GasFeeCap())
		result.GasTipCap = (*hexutil.Big)(tx.GasTipCap())
		result.GasPrice = nil
	}

	if tx.ChainId() != nil {
		result.ChainID = (*hexutil.Big)(tx.ChainId())
	}

	return result
}

// NewRPCPendingTransaction creates an RPCTransaction for a pending transaction
func NewRPCPendingTransaction(tx *types.Transaction) *RPCTransaction {
	return NewRPCTransaction(tx, common.Hash{}, 0, 0)
}

// RPCReceipt represents a transaction receipt in RPC format
type RPCReceipt struct {
	TransactionHash   common.Hash     `json:"transactionHash"`
	TransactionIndex  hexutil.Uint64  `json:"transactionIndex"`
	BlockHash         common.Hash     `json:"blockHash"`
	BlockNumber       *hexutil.Big    `json:"blockNumber"`
	From              common.Address  `json:"from"`
	To                *common.Address `json:"to"`
	CumulativeGasUsed hexutil.Uint64  `json:"cumulativeGasUsed"`
	GasUsed           hexutil.Uint64  `json:"gasUsed"`
	ContractAddress   *common.Address `json:"contractAddress"`
	Logs              []*types.Log    `json:"logs"`
	LogsBloom         types.Bloom     `json:"logsBloom"`
	Type              hexutil.Uint64  `json:"type"`
	Status            hexutil.Uint64  `json:"status"`
	EffectiveGasPrice *hexutil.Big    `json:"effectiveGasPrice,omitempty"`
}

// NewRPCReceipt creates an RPCReceipt from a types.Receipt
func NewRPCReceipt(receipt *types.Receipt, tx *types.Transaction, blockHash common.Hash, blockNumber uint64, index uint64) *RPCReceipt {
	from, _ := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)

	rpcReceipt := &RPCReceipt{
		TransactionHash:   tx.Hash(),
		TransactionIndex:  hexutil.Uint64(index),
		BlockHash:         blockHash,
		BlockNumber:       (*hexutil.Big)(big.NewInt(int64(blockNumber))),
		From:              from,
		To:                tx.To(),
		CumulativeGasUsed: hexutil.Uint64(receipt.CumulativeGasUsed),
		GasUsed:           hexutil.Uint64(receipt.GasUsed),
		ContractAddress:   nil,
		Logs:              receipt.Logs,
		LogsBloom:         receipt.Bloom,
		Type:              hexutil.Uint64(tx.Type()),
		Status:            hexutil.Uint64(receipt.Status),
	}

	if receipt.Logs == nil {
		rpcReceipt.Logs = []*types.Log{}
	}

	// Set contract address if this is a contract creation
	if tx.To() == nil && len(receipt.ContractAddress) > 0 {
		rpcReceipt.ContractAddress = &receipt.ContractAddress
	}

	// Calculate effective gas price
	if tx.Type() == types.DynamicFeeTxType {
		if receipt.EffectiveGasPrice != nil {
			rpcReceipt.EffectiveGasPrice = (*hexutil.Big)(receipt.EffectiveGasPrice)
		}
	} else {
		rpcReceipt.EffectiveGasPrice = (*hexutil.Big)(tx.GasPrice())
	}

	return rpcReceipt
}

// FeeHistoryResult represents the result of eth_feeHistory
type FeeHistoryResult struct {
	OldestBlock  *hexutil.Big     `json:"oldestBlock"`
	BaseFeePerGas []*hexutil.Big   `json:"baseFeePerGas,omitempty"`
	GasUsedRatio []float64        `json:"gasUsedRatio"`
	Reward       [][]*hexutil.Big `json:"reward,omitempty"`
}

// CallArgs represents the arguments for a call
type CallArgs struct {
	From                 *common.Address `json:"from"`
	To                   *common.Address `json:"to"`
	Gas                  *hexutil.Uint64 `json:"gas"`
	GasPrice             *hexutil.Big    `json:"gasPrice"`
	MaxFeePerGas         *hexutil.Big    `json:"maxFeePerGas"`
	MaxPriorityFeePerGas *hexutil.Big    `json:"maxPriorityFeePerGas"`
	Value                *hexutil.Big    `json:"value"`
	Data                 *hexutil.Bytes  `json:"data"`
}
