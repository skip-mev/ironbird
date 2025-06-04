package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/skip-mev/ironbird/server"
	"github.com/skip-mev/ironbird/server/db"
	"github.com/skip-mev/ironbird/types"
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))
}

func main() {
	logger, _ := zap.NewDevelopment()

	grpcAddrFlag := flag.String("grpc-addr", ":50051", "gRpc server address")
	flag.Parse()

	dbPath := getEnvOrDefault("DATABASE_PATH", "./ironbird.db")
	logger.Info("Connecting to database", zap.String("path", dbPath))

	database, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	migrationsPath := "./migrations"
	if err := database.RunMigrations(migrationsPath); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	logger.Info("Database initialized successfully")

	temporalConfig := types.TemporalConfig{
		Host:      getEnvOrDefault("TEMPORAL_HOST", "127.0.0.1:7233"),
		Namespace: getEnvOrDefault("TEMPORAL_NAMESPACE", "default"),
	}

	grpcServer, err := server.NewGRpcServer(temporalConfig, database, logger)
	if err != nil {
		logger.Error("creating gRpc server", zap.Error(err))
		os.Exit(1)
	}

	go func() {
		logger.Info("starting gRpc server", zap.String("address", *grpcAddrFlag))
		if err := grpcServer.Start(*grpcAddrFlag); err != nil {
			logger.Error("starting gRpc server", zap.Error(err))
			os.Exit(1)
		}
	}()

	logger.Info("server started successfully")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	logger.Info("received signal, shutting down", zap.String("signal", sig.String()))

	grpcServer.Stop()

	logger.Info("server shutdown complete")
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
