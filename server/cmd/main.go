package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"context"
	"log"

	"github.com/skip-mev/ironbird/core/db"
	"github.com/skip-mev/ironbird/core/types"
	"github.com/skip-mev/ironbird/server"
	"go.uber.org/zap"
	"google.golang.org/grpc/grpclog"
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))
}

func main() {
	logger, _ := zap.NewDevelopment()

	grpcAddrFlag := flag.String("grpc-addr", ":50051", "gRPC server address")
	flag.Parse()

	dbPath := getEnvOrDefault("DATABASE_PATH", "./ironbird.db")
	database, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		logger.Error("connecting to database", zap.Error(err))
		os.Exit(1)
	}

	temporalConfig := types.TemporalConfig{
		Host:      getEnvOrDefault("TEMPORAL_HOST", "127.0.0.1:7233"),
		Namespace: getEnvOrDefault("TEMPORAL_NAMESPACE", "default"),
	}

	grpcServer, err := server.NewGRPCServer(temporalConfig, database, logger)
	if err != nil {
		logger.Error("creating gRPC server", zap.Error(err))
		os.Exit(1)
	}

	go func() {
		logger.Info("starting gRPC server", zap.String("address", *grpcAddrFlag))
		if err := grpcServer.Start(*grpcAddrFlag); err != nil {
			logger.Error("starting gRPC server", zap.Error(err))
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
