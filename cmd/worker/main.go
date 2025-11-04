package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/skip-mev/ironbird/activities/builder"
	"github.com/skip-mev/ironbird/activities/loadbalancer"
	"github.com/skip-mev/ironbird/activities/loadtest"
	testnetactivity "github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
	pb "github.com/skip-mev/ironbird/server/proto"
	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/util"
	testnetworkflow "github.com/skip-mev/ironbird/workflows/testnet"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"go.temporal.io/sdk/worker"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	configFlag = flag.String("config", "./conf/worker.yaml", "Path to the worker configuration file")
)

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	flag.Parse()

	cfg, err := types.ParseWorkerConfig(*configFlag)

	if err != nil {
		panic(err)
	}

	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
		MetricsHandler: sdktally.NewMetricsHandler(util.NewPrometheusScope(prometheus.Configuration{
			ListenAddress: "0.0.0.0:9092",
			TimerType:     "histogram",
		})),
	})

	if err != nil {
		log.Fatalln(err)
	}

	defer c.Close()

	activeRegistry := cfg.Builder.GetActiveRegistry()
	logger.Info("Using registry configuration",
		zap.String("mode", activeRegistry.Type),
		zap.String("image_name", activeRegistry.ImageName))

	var awsConfig *aws.Config
	if activeRegistry.Type == "ecr" {
		awsCfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			log.Fatalln("Failed to load AWS config (required for ECR registry):", err)
		}
		awsConfig = &awsCfg
		logger.Info("AWS config loaded for ECR registry")
	} else {
		logger.Info("Skipping AWS config (using local registry)")
	}

	builderActivity := builder.Activity{
		BuilderConfig: cfg.Builder,
		AwsConfig:     awsConfig,
		Chains:        cfg.Chains,
		Registry:      activeRegistry,
	}

	var grpcClient pb.IronbirdServiceClient
	if cfg.ServerAddress != "" {
		logger.Info("Attempting to connect to gRPC server", zap.String("address", cfg.ServerAddress))

		conn, err := grpc.NewClient(cfg.ServerAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			logger.Error("Failed to connect to server", zap.String("address", cfg.ServerAddress), zap.Error(err))
			logger.Warn("Continuing without gRPC client - workflow data updates will be skipped")
		} else {
			grpcClient = pb.NewIronbirdServiceClient(conn)
			logger.Info("Successfully connected to gRPC server", zap.String("address", cfg.ServerAddress))

			defer func() {
				if closeErr := conn.Close(); closeErr != nil {
					logger.Warn("Error closing gRPC connection", zap.Error(closeErr))
				}
			}()
		}
	} else {
		logger.Warn("no grpc client configured - workflow data updates will be skipped")
	}

	var tailscaleSettings digitalocean.TailscaleSettings
	if cfg.Tailscale.ServerOauthSecret != "" && cfg.Tailscale.NodeAuthKey != "" {
		var err error
		tailscaleSettings, err = digitalocean.SetupTailscale(ctx, cfg.Tailscale.ServerOauthSecret,
			cfg.Tailscale.NodeAuthKey, "ironbird", cfg.Tailscale.ServerTags, cfg.Tailscale.NodeTags)
		if err != nil {
			panic(err)
		}
		logger.Info("Tailscale configured successfully")
	} else {
		logger.Info("Skipping Tailscale setup (credentials not provided - required for DigitalOcean workflows)")
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
		Pyroscope: digitalocean.PyroscopeSettings{
			URL: cfg.Telemetry.Pyroscope.URL,
		},
	}

	testnetActivity := testnetactivity.Activity{
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
		DOToken:           cfg.DigitalOcean.Token,
		Chains:            cfg.Chains,
		GrafanaConfig:     cfg.Grafana,
		GRPCClient:        grpcClient,
		AwsConfig:         awsConfig,
		RegistryType:      activeRegistry.Type,
	}

	loadTestActivity := loadtest.Activity{
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
	}

	var sslKey, sslCert []byte
	if cfg.LoadBalancer.SSLKeyPath != "" && cfg.LoadBalancer.SSLCertPath != "" {
		var err error
		sslKey, err = os.ReadFile(cfg.LoadBalancer.SSLKeyPath)
		if err != nil {
			log.Printf("Failed to read SSL key from path %s: %v", cfg.LoadBalancer.SSLKeyPath, err)
			os.Exit(1)
		}

		sslCert, err = os.ReadFile(cfg.LoadBalancer.SSLCertPath)
		if err != nil {
			log.Printf("Failed to read SSL certificate from path %s: %v", cfg.LoadBalancer.SSLCertPath, err)
			os.Exit(1)
		}
		logger.Info("SSL certificates loaded for load balancer")
	} else {
		logger.Info("Skipping SSL certificate loading (not configured - required for DigitalOcean load balancer)")
	}

	loadBalancerActivity := loadbalancer.Activity{
		RootDomain:        cfg.LoadBalancer.RootDomain,
		SSLKey:            sslKey,
		SSLCertificate:    sslCert,
		DOToken:           cfg.DigitalOcean.Token,
		TailscaleSettings: tailscaleSettings,
		TelemetrySettings: telemetrySettings,
		GRPCClient:        grpcClient,
	}

	w := worker.New(c, messages.TaskQueue, worker.Options{})

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
