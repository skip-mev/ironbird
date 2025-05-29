package main

import (
	"context"
	"flag"
	database_service "github.com/skip-mev/ironbird/database"
	"os"

	"github.com/skip-mev/ironbird/activities/loadbalancer"
	"github.com/skip-mev/ironbird/db"
	"github.com/skip-mev/ironbird/util"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.uber.org/zap"

	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/skip-mev/petri/core/v3/provider/digitalocean"
	"github.com/uber-go/tally/v4/prometheus"

	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/loadtest"
	testnetactivity "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/types"
	testnetworkflow "github.com/skip-mev/ironbird/workflows/testnet"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

var (
	configFlag = flag.String("config", "./conf/worker.yaml", "Path to the worker configuration file")
	chainsFlag = flag.String("chains", "./conf/chains.yaml", "Path to the chain images configuration file")
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	flag.Parse()

	dbPath := getEnvOrDefault("DATABASE_PATH", "./ironbird.db")
	logger.Info("Connecting to database", zap.String("path", dbPath))

	database, err := db.NewSQLiteDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	migrationsPath := "./migrations"
	if err := database.RunMigrations(migrationsPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	logger.Info("Database initialized successfully")

	databaseService := database_service.NewDatabaseService(database, logger)

	cfg, err := types.ParseWorkerConfig(*configFlag)

	if err != nil {
		panic(err)
	}

	chainImages, err := types.ParseChainImagesConfig(*chainsFlag)
	if err != nil {
		panic(err)
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9091",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	awsConfig, err := config.LoadDefaultConfig(ctx)

	if err != nil {
		log.Fatalln(err)
	}

	builderActivity := builder.Activity{BuilderConfig: cfg.Builder, AwsConfig: &awsConfig, ChainImages: chainImages}

	tailscaleSettings, err := digitalocean.SetupTailscale(ctx, cfg.Tailscale.ServerOauthSecret,
		cfg.Tailscale.NodeAuthKey, "ironbird", cfg.Tailscale.ServerTags, cfg.Tailscale.NodeTags)
	if err != nil {
		panic(err)
	}

	telemetrySettings := digitalocean.TelemetrySettings{
		Prometheus: digitalocean.PrometheusSettings{
			URL:      cfg.Telemetry.Prometheus.URL,
			Username: cfg.Telemetry.Prometheus.Username,
			Password: cfg.Telemetry.Prometheus.Password,
		},
		Loki: digitalocean.LokiSettings{
			URL:      cfg.Telemetry.Loki.URL,
			Username: cfg.Telemetry.Loki.Username,
			Password: cfg.Telemetry.Loki.Password,
		},
	}

	testnetActivity := testnetactivity.Activity{
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
		DOToken:           cfg.DigitalOcean.Token,
		ChainImages:       chainImages,
		DatabaseService:   databaseService,
	}

	loadTestActivity := loadtest.Activity{
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
	}

	sslKey, err := os.ReadFile(cfg.LoadBalancer.SSLKeyPath)

	if err != nil {
		log.Printf("Failed to read SSL key from path %s: %v", cfg.LoadBalancer.SSLKeyPath, err)
		os.Exit(1)
	}

	sslCert, err := os.ReadFile(cfg.LoadBalancer.SSLCertPath)

	if err != nil {
		log.Printf("Failed to read SSL certificate from path %s: %v", cfg.LoadBalancer.SSLCertPath, err)
		os.Exit(1)
	}

	loadBalancerActivity := loadbalancer.Activity{
		RootDomain:        cfg.LoadBalancer.RootDomain,
		SSLKey:            sslKey,
		SSLCertificate:    sslCert,
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
	}

	w := worker.New(c, testnetworkflow.TaskQueue, worker.Options{})

	w.RegisterWorkflow(testnetworkflow.Workflow)

	w.RegisterActivity(testnetActivity.LaunchTestnet)
	w.RegisterActivity(testnetActivity.CreateProvider)
	w.RegisterActivity(testnetActivity.TeardownProvider)
	w.RegisterActivity(loadTestActivity.RunLoadTest)
	w.RegisterActivity(loadBalancerActivity.LaunchLoadBalancer)
	w.RegisterActivity(builderActivity.BuildDockerImage)

	err = w.Run(worker.InterruptCh())

	if err != nil {
		log.Fatalln(err)
	}
}
