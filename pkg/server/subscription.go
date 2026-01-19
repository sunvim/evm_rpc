package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

// SubscriptionType represents the type of subscription
type SubscriptionType string

const (
	SubscriptionNewHeads              SubscriptionType = "newHeads"
	SubscriptionLogs                  SubscriptionType = "logs"
	SubscriptionNewPendingTransactions SubscriptionType = "newPendingTransactions"
)

// Subscription represents a client subscription
type Subscription struct {
	ID       string
	Type     SubscriptionType
	Filter   *FilterCriteria
	conn     *WebSocketConnection
	cancelFn context.CancelFunc
}

// FilterCriteria represents log filter criteria
type FilterCriteria struct {
	Addresses []common.Address `json:"address,omitempty"`
	Topics    [][]common.Hash  `json:"topics,omitempty"`
}

// SubscriptionManager manages client subscriptions
type SubscriptionManager struct {
	mu            sync.RWMutex
	subscriptions map[string]*Subscription // subscription ID -> subscription
	connections   map[*WebSocketConnection]map[string]*Subscription // conn -> subscription IDs
	pikaClient    *storage.PikaClient
	blockReader   *storage.BlockReader
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(pikaClient *storage.PikaClient, blockReader *storage.BlockReader) *SubscriptionManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	sm := &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
		connections:   make(map[*WebSocketConnection]map[string]*Subscription),
		pikaClient:    pikaClient,
		blockReader:   blockReader,
		ctx:           ctx,
		cancel:        cancel,
	}

	// Start subscription workers
	sm.wg.Add(2)
	go sm.listenNewBlocks()
	go sm.listenNewPendingTransactions()

	return sm
}

// Subscribe creates a new subscription
func (sm *SubscriptionManager) Subscribe(conn *WebSocketConnection, subType SubscriptionType, filter *FilterCriteria) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate subscription ID
	subID := generateSubscriptionID()

	// Create subscription context
	_, cancel := context.WithCancel(sm.ctx)

	sub := &Subscription{
		ID:       subID,
		Type:     subType,
		Filter:   filter,
		conn:     conn,
		cancelFn: cancel,
	}

	// Store subscription
	sm.subscriptions[subID] = sub

	// Store connection mapping
	if sm.connections[conn] == nil {
		sm.connections[conn] = make(map[string]*Subscription)
	}
	sm.connections[conn][subID] = sub

	// Update metrics
	metrics.RecordSubscription(string(subType), 1)

	logger.Infof("Created subscription: id=%s, type=%s", subID, subType)

	return subID, nil
}

// Unsubscribe removes a subscription
func (sm *SubscriptionManager) Unsubscribe(subID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sub, exists := sm.subscriptions[subID]
	if !exists {
		return fmt.Errorf("subscription not found: %s", subID)
	}

	// Cancel subscription context
	if sub.cancelFn != nil {
		sub.cancelFn()
	}

	// Remove from subscriptions
	delete(sm.subscriptions, subID)

	// Remove from connection mapping
	if connSubs, ok := sm.connections[sub.conn]; ok {
		delete(connSubs, subID)
		if len(connSubs) == 0 {
			delete(sm.connections, sub.conn)
		}
	}

	// Update metrics
	metrics.RecordSubscription(string(sub.Type), -1)

	logger.Infof("Removed subscription: id=%s, type=%s", subID, sub.Type)

	return nil
}

// UnsubscribeAll removes all subscriptions for a connection
func (sm *SubscriptionManager) UnsubscribeAll(conn *WebSocketConnection) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	connSubs, exists := sm.connections[conn]
	if !exists {
		return
	}

	// Cancel and remove all subscriptions for this connection
	for subID, sub := range connSubs {
		if sub.cancelFn != nil {
			sub.cancelFn()
		}
		delete(sm.subscriptions, subID)
		metrics.RecordSubscription(string(sub.Type), -1)
	}

	delete(sm.connections, conn)

	logger.Infof("Removed all subscriptions for connection")
}

// listenNewBlocks listens for new blocks from Pika pub/sub
func (sm *SubscriptionManager) listenNewBlocks() {
	defer sm.wg.Done()

	// Subscribe to Pika channel
	pubsub := sm.pikaClient.Subscribe(sm.ctx, "blocks:new")
	defer pubsub.Close()

	logger.Info("Listening for new blocks...")

	for {
		select {
		case <-sm.ctx.Done():
			return
		default:
			msg, err := pubsub.ReceiveMessage(sm.ctx)
			if err != nil {
				if sm.ctx.Err() != nil {
					return
				}
				logger.Errorf("Failed to receive block message: %v", err)
				continue
			}

			// Parse block hash
			blockHash := common.HexToHash(msg.Payload)
			
			// Get full block
			block, err := sm.blockReader.GetBlockByHash(sm.ctx, blockHash)
			if err != nil {
				logger.Errorf("Failed to get block: %v", err)
				continue
			}

			// Notify subscribers
			sm.notifyNewHeads(block)
			sm.notifyLogs(block)
		}
	}
}

// listenNewPendingTransactions listens for new pending transactions from Pika pub/sub
func (sm *SubscriptionManager) listenNewPendingTransactions() {
	defer sm.wg.Done()

	// Subscribe to Pika channel
	pubsub := sm.pikaClient.Subscribe(sm.ctx, "pool:new")
	defer pubsub.Close()

	logger.Info("Listening for new pending transactions...")

	for {
		select {
		case <-sm.ctx.Done():
			return
		default:
			msg, err := pubsub.ReceiveMessage(sm.ctx)
			if err != nil {
				if sm.ctx.Err() != nil {
					return
				}
				logger.Errorf("Failed to receive tx message: %v", err)
				continue
			}

			// Parse transaction hash
			txHash := common.HexToHash(msg.Payload)
			
			// Notify subscribers
			sm.notifyNewPendingTransaction(txHash)
		}
	}
}

// notifyNewHeads notifies newHeads subscribers
func (sm *SubscriptionManager) notifyNewHeads(block *types.Block) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	header := block.Header()

	for _, sub := range sm.subscriptions {
		if sub.Type != SubscriptionNewHeads {
			continue
		}

		// Create notification
		notification := map[string]interface{}{
			"subscription": sub.ID,
			"result": map[string]interface{}{
				"number":     fmt.Sprintf("0x%x", header.Number.Uint64()),
				"hash":       header.Hash().Hex(),
				"parentHash": header.ParentHash.Hex(),
				"timestamp":  fmt.Sprintf("0x%x", header.Time),
				"gasUsed":    fmt.Sprintf("0x%x", header.GasUsed),
				"gasLimit":   fmt.Sprintf("0x%x", header.GasLimit),
			},
		}

		// Send notification
		if err := sub.conn.SendNotification(notification); err != nil {
			logger.Errorf("Failed to send newHeads notification: %v", err)
		} else {
			metrics.RecordNotification(string(SubscriptionNewHeads))
		}
	}
}

// notifyLogs notifies logs subscribers
func (sm *SubscriptionManager) notifyLogs(block *types.Block) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Get receipts for block
	receipts, err := sm.blockReader.GetReceipts(sm.ctx, block.NumberU64())
	if err != nil {
		logger.Errorf("Failed to get receipts: %v", err)
		return
	}

	// Extract logs
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			sm.notifyLog(log)
		}
	}
}

// notifyLog notifies subscribers about a specific log
func (sm *SubscriptionManager) notifyLog(log *types.Log) {
	for _, sub := range sm.subscriptions {
		if sub.Type != SubscriptionLogs {
			continue
		}

		// Check filter criteria
		if sub.Filter != nil {
			if !matchLogFilter(log, sub.Filter) {
				continue
			}
		}

		// Create notification
		notification := map[string]interface{}{
			"subscription": sub.ID,
			"result": map[string]interface{}{
				"address":          log.Address.Hex(),
				"topics":           log.Topics,
				"data":             fmt.Sprintf("0x%x", log.Data),
				"blockNumber":      fmt.Sprintf("0x%x", log.BlockNumber),
				"transactionHash":  log.TxHash.Hex(),
				"transactionIndex": fmt.Sprintf("0x%x", log.TxIndex),
				"blockHash":        log.BlockHash.Hex(),
				"logIndex":         fmt.Sprintf("0x%x", log.Index),
			},
		}

		// Send notification
		if err := sub.conn.SendNotification(notification); err != nil {
			logger.Errorf("Failed to send logs notification: %v", err)
		} else {
			metrics.RecordNotification(string(SubscriptionLogs))
		}
	}
}

// notifyNewPendingTransaction notifies newPendingTransactions subscribers
func (sm *SubscriptionManager) notifyNewPendingTransaction(txHash common.Hash) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for _, sub := range sm.subscriptions {
		if sub.Type != SubscriptionNewPendingTransactions {
			continue
		}

		// Create notification
		notification := map[string]interface{}{
			"subscription": sub.ID,
			"result":       txHash.Hex(),
		}

		// Send notification
		if err := sub.conn.SendNotification(notification); err != nil {
			logger.Errorf("Failed to send newPendingTransactions notification: %v", err)
		} else {
			metrics.RecordNotification(string(SubscriptionNewPendingTransactions))
		}
	}
}

// matchLogFilter checks if a log matches filter criteria
func matchLogFilter(log *types.Log, filter *FilterCriteria) bool {
	// Check addresses
	if len(filter.Addresses) > 0 {
		matched := false
		for _, addr := range filter.Addresses {
			if log.Address == addr {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check topics
	for i, topicSet := range filter.Topics {
		if i >= len(log.Topics) {
			return false
		}
		if len(topicSet) == 0 {
			continue // wildcard
		}
		matched := false
		for _, topic := range topicSet {
			if log.Topics[i] == topic {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	return true
}

// Stop stops the subscription manager
func (sm *SubscriptionManager) Stop() {
	logger.Info("Stopping subscription manager...")
	sm.cancel()
	sm.wg.Wait()
	logger.Info("Subscription manager stopped")
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	// Generate a random hex string
	return fmt.Sprintf("0x%x", common.BigToHash(common.Big1).Bytes()[:16])
}
