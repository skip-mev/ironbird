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

var (
	config = flag.String("config", "./conf/server.yaml", "Path to the server configuration file")
)

func init() {
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(os.Stdout, os.Stdout, os.Stderr))
}

func main() {
	logger, _ := zap.NewDevelopment()
	flag.Parse()

	cfg, err := types.ParseServerConfig(*config)

	if err != nil {
		panic(err)
	}

	database, err := db.NewSQLiteDB(cfg.DatabasePath, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer database.Close()

	if err := database.RunMigrations(cfg.MigrationsPath); err != nil {
		logger.Fatal("Failed to run migrations", zap.Error(err))
	}

	logger.Info("Database initialized successfully")

	temporalConfig := types.TemporalConfig{
		Host:      cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	}

	grpcServer, err := server.NewGRPCServer(temporalConfig, database, logger)
	if err != nil {
		logger.Error("creating gRpc server", zap.Error(err))
		os.Exit(1)
	}

	go func() {
		logger.Info("starting gRpc server", zap.String("address", cfg.GrpcAddress))
		if err := grpcServer.Start(cfg.GrpcAddress, cfg.GrpcWebAddress); err != nil {
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
