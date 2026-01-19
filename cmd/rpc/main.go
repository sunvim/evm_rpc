package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sunvim/evm_rpc/pkg/api/eth"
	"github.com/sunvim/evm_rpc/pkg/api/net"
	"github.com/sunvim/evm_rpc/pkg/api/txpool"
	"github.com/sunvim/evm_rpc/pkg/api/web3"
	"github.com/sunvim/evm_rpc/pkg/cache"
	"github.com/sunvim/evm_rpc/pkg/config"
	"github.com/sunvim/evm_rpc/pkg/logger"
	"github.com/sunvim/evm_rpc/pkg/metrics"
	"github.com/sunvim/evm_rpc/pkg/middleware"
	"github.com/sunvim/evm_rpc/pkg/server"
	"github.com/sunvim/evm_rpc/pkg/storage"
)

var (
	version = "v1.0.0"
	commit  = "unknown"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config/config.yaml", "Path to configuration file")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("EVM RPC Service %s (commit: %s)\n", version, commit)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfigWithDefaults(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.InitLogger(cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Infof("Starting EVM RPC Service %s", version)
	logger.Infof("Chain: %s (ID: %d)", cfg.Chain.Name, cfg.Chain.ChainID)

	// Initialize Pika client
	logger.Info("Connecting to Pika storage...")
	pikaClient, err := storage.NewPikaClient(cfg.Storage.Pika)
	if err != nil {
		logger.Fatalf("Failed to connect to Pika: %v", err)
	}
	defer pikaClient.Close()
	logger.Info("Connected to Pika storage")

	// Initialize storage readers
	blockReader := storage.NewBlockReader(pikaClient)
	txReader := storage.NewTransactionReader(pikaClient)
	stateReader := storage.NewStateReader(pikaClient)
	txPoolStorage := storage.NewTxPoolStorage(pikaClient)

	// Initialize cache manager
	var cacheManager *cache.Manager
	if cfg.Cache.Enabled {
		logger.Info("Initializing cache manager...")
		cacheManager, err = cache.NewManager(cfg.Cache)
		if err != nil {
			logger.Fatalf("Failed to initialize cache: %v", err)
		}
		logger.Info("Cache manager initialized")
	}

	// Initialize API handlers
	logger.Info("Initializing API handlers...")
	blockAPI := eth.NewBlockAPI(blockReader, cfg.Chain.ChainID)
	gasAPI := eth.NewGasAPI(blockReader, cfg.Chain.ChainID)
	stateAPI := eth.NewStateAPI(blockReader, stateReader, cfg.Chain.ChainID)
	txAPI := eth.NewTransactionAPI(blockReader, txReader, cfg.Chain.ChainID)
	txPoolAPI := eth.NewTxPoolAPI(blockReader, stateReader, txPoolStorage, cfg.Chain.ChainID)
	netAPI := net.NewNetAPI(cfg.Chain.NetworkID)
	web3API := web3.NewWeb3API(version)
	txpoolNS := txpool.NewTxPoolAPI(txPoolStorage)

	// Initialize JSON-RPC handler
	var rateLimiter *middleware.RateLimiter
	if cfg.RateLimit.Enabled {
		logger.Info("Initializing rate limiter...")
		rateLimiter = middleware.NewRateLimiter(
			cfg.RateLimit.Enabled,
			cfg.RateLimit.Global.RequestsPerSecond,
			cfg.RateLimit.Global.Burst,
			cfg.RateLimit.IP.RequestsPerSecond,
			cfg.RateLimit.IP.Burst,
			cfg.RateLimit.Method,
		)
		logger.Info("Rate limiter initialized")
	}

	rpcHandler := server.NewJSONRPCHandler(rateLimiter, cfg.Logging.SlowQueryThreshold)

	// Register API services with their namespaces
	if err := rpcHandler.RegisterService("eth", blockAPI); err != nil {
		logger.Fatalf("Failed to register block API: %v", err)
	}
	if err := rpcHandler.RegisterService("eth", gasAPI); err != nil {
		logger.Fatalf("Failed to register gas API: %v", err)
	}
	if err := rpcHandler.RegisterService("eth", stateAPI); err != nil {
		logger.Fatalf("Failed to register state API: %v", err)
	}
	if err := rpcHandler.RegisterService("eth", txAPI); err != nil {
		logger.Fatalf("Failed to register transaction API: %v", err)
	}
	if err := rpcHandler.RegisterService("eth", txPoolAPI); err != nil {
		logger.Fatalf("Failed to register tx pool API: %v", err)
	}
	if err := rpcHandler.RegisterService("net", netAPI); err != nil {
		logger.Fatalf("Failed to register net API: %v", err)
	}
	if err := rpcHandler.RegisterService("web3", web3API); err != nil {
		logger.Fatalf("Failed to register web3 API: %v", err)
	}
	if err := rpcHandler.RegisterService("txpool", txpoolNS); err != nil {
		logger.Fatalf("Failed to register txpool API: %v", err)
	}

	// Initialize metrics
	if cfg.Metrics.Enabled {
		logger.Infof("Starting metrics server on %s", cfg.Metrics.ListenAddr)
		metricsServer := metrics.NewServer(cfg.Metrics.ListenAddr)
		go func() {
			if err := metricsServer.Start(); err != nil {
				logger.Errorf("Metrics server error: %v", err)
			}
		}()
	}

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize subscription manager for WebSocket
	var subManager *server.SubscriptionManager
	if cfg.Server.WS.Enabled {
		logger.Info("Initializing subscription manager...")
		subManager = server.NewSubscriptionManager(pikaClient, blockReader)
		// Subscription manager doesn't have a Run method - it starts listening internally
		logger.Info("Subscription manager initialized")
	}

	// Create middleware
	loggingMiddleware := middleware.NewLoggingMiddleware(cfg.Logging.SlowQueryThreshold)
	corsMiddleware := middleware.NewCORS(cfg.Server.HTTP.CORSOrigins)

	// Initialize HTTP server
	var httpServer *server.HTTPServer
	if cfg.Server.HTTP.Enabled {
		logger.Infof("Initializing HTTP server on %s", cfg.Server.HTTP.ListenAddr)
		httpServer = server.NewHTTPServer(
			cfg.Server.HTTP,
			rpcHandler,
			blockReader,
			rateLimiter,
			loggingMiddleware,
			corsMiddleware,
		)
	}

	// Initialize WebSocket server
	var wsServer *server.WebSocketServer
	if cfg.Server.WS.Enabled {
		logger.Infof("Initializing WebSocket server on %s", cfg.Server.WS.ListenAddr)
		wsServer = server.NewWebSocketServer(
			cfg.Server.WS,
			rpcHandler,
			subManager,
			cfg.Server.HTTP.CORSOrigins,
		)
	}

	// Start servers
	errChan := make(chan error, 2)

	if httpServer != nil {
		go func() {
			logger.Infof("Starting HTTP server on %s", cfg.Server.HTTP.ListenAddr)
			if err := httpServer.Start(); err != nil {
				errChan <- fmt.Errorf("HTTP server error: %w", err)
			}
		}()
	}

	if wsServer != nil {
		go func() {
			logger.Infof("Starting WebSocket server on %s", cfg.Server.WS.ListenAddr)
			if err := wsServer.Start(); err != nil {
				errChan <- fmt.Errorf("WebSocket server error: %w", err)
			}
		}()
	}

	logger.Info("All servers started successfully")

	// Log cache statistics periodically
	if cacheManager != nil {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					stats := cacheManager.Stats()
					for name, stat := range stats {
						logger.Infof("Cache[%s] - Hits: %d, Misses: %d, Size: %d, HitRate: %.2f%%",
							name, stat.Hits, stat.Misses, stat.Size, stat.HitRate*100)
					}
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Errorf("Server error: %v", err)
	case sig := <-sigChan:
		logger.Infof("Received signal: %v", sig)
	}

	// Graceful shutdown
	logger.Info("Shutting down servers...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if httpServer != nil {
		if err := httpServer.Stop(shutdownCtx); err != nil {
			logger.Errorf("HTTP server shutdown error: %v", err)
		}
	}

	if wsServer != nil {
		if err := wsServer.Stop(shutdownCtx); err != nil {
			logger.Errorf("WebSocket server shutdown error: %v", err)
		}
	}

	logger.Info("Shutdown complete")
}
