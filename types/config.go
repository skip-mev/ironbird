package types

import (
	"fmt"
	"os"
	"time"

	petritypes "github.com/skip-mev/ironbird/petri/core/types"
	petrichain "github.com/skip-mev/ironbird/petri/cosmos/chain"

	"gopkg.in/yaml.v3"
)

type TailscaleConfig struct {
	ServerOauthSecret string
	ServerTags        []string `yaml:"server_tags"`
	NodeAuthKey       string
	NodeTags          []string `yaml:"node_tags"`
}

type WorkerConfig struct {
	Temporal      TemporalConfig     `yaml:"temporal"`
	Tailscale     TailscaleConfig    `yaml:"tailscale"`
	DigitalOcean  DigitalOceanConfig `yaml:"digitalocean"`
	LoadBalancer  LoadBalancerConfig `yaml:"load_balancer"`
	Telemetry     TelemetryConfig    `yaml:"telemetry"`
	Builder       BuilderConfig      `yaml:"builder"`
	Chains        Chains             `yaml:"chains"`
	Grafana       GrafanaConfig      `yaml:"grafana"`
	ServerAddress string             `yaml:"server_address"`
}

type LoadBalancerConfig struct {
	RootDomain  string `yaml:"root_domain"`
	SSLKeyPath  string `yaml:"ssl_key_path"`
	SSLCertPath string `yaml:"ssl_cert_path"`
}

type TelemetryConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Loki       LokiConfig       `yaml:"loki"`
	Pyroscope  PyroscopeConfig  `yaml:"pyroscope"`
}

type PyroscopeConfig struct {
	URL string `json:"url"`
}

type PrometheusConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type LokiConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type GrafanaDashboard struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	HumanName string `yaml:"human_name"`
}

type TemporalConfig struct {
	Host      string `yaml:"host"`
	Namespace string `yaml:"namespace,omitempty"`
}

type DigitalOceanConfig struct {
	Token string `yaml:"token"`
}

type BuilderConfig struct {
	BuildKitAddress string            `yaml:"build_kit_address"`
	Registry        RegistryConfig    `yaml:"registry"`
	AuthEnvConfigs  map[string]string `yaml:"auth_env_configs"`
}

type RegistryConfig struct {
	// e.g. <account_id>.dkr.ecr.<region>.amazonaws.com
	URL string `yaml:"url"`

	// e.g. skip-mev/ironbird
	ImageName string `yaml:"image_name"`
}
type ChainsConfig struct {
	Name                  string                    `yaml:"name"`
	Image                 string                    `yaml:"image"`
	GenesisModifications  []petrichain.GenesisKV    `yaml:"genesis_modifications"`
	NumOfNodes            uint64                    `yaml:"num_of_nodes"`
	NumOfValidators       uint64                    `yaml:"num_of_validators"`
	RegionConfigs         []petritypes.RegionConfig `yaml:"region_configs"`
	CustomAppConfig       map[string]interface{}    `yaml:"custom_app_config"`
	CustomConsensusConfig map[string]interface{}    `yaml:"custom_consensus_config"`
	CustomClientConfig    map[string]interface{}    `yaml:"custom_client_config"`
	SetSeedNode           bool                      `yaml:"set_seed_node"`
	SetPersistentPeers    bool                      `yaml:"set_persistent_peers"`
}

type GrafanaConfig struct {
	URL        string      `yaml:"url"`
	Dashboards []Dashboard `yaml:"dashboards"`
}

type Dashboard struct {
	ID        string `yaml:"id"`
	Name      string `yaml:"name"`
	HumanName string `yaml:"human_name"`
}

func GenerateMonitoringLinks(chainID string, startTime time.Time, grafana GrafanaConfig) map[string]string {
	urls := make(map[string]string)

	for _, dashboard := range grafana.Dashboards {
		url := fmt.Sprintf("%s/d/%s/%s?orgId=1&var-chain_id=%s&from=%d&to=%s&refresh=auto",
			grafana.URL, dashboard.ID, dashboard.Name, chainID, startTime.UnixMilli(), "now")
		urls[dashboard.HumanName] = url
	}

	return urls
}

type ServerConfig struct {
	Temporal       TemporalConfig `yaml:"temporal"`
	DatabasePath   string         `yaml:"database_path"`
	MigrationsPath string         `yaml:"migrations_path"`
	GrpcAddress    string         `yaml:"grpc_address"`
	GrpcWebAddress string         `yaml:"grpc_web_address"`
}

type Chains map[string]ImageConfig

type ImageConfig struct {
	Name            string   `yaml:"name"`
	Version         string   `yaml:"version"`
	Dockerfile      string   `yaml:"dockerfile"`
	AdditionalFiles []string `yaml:"additional_files"`
	GID             string   `yaml:"gid"`
	UID             string   `yaml:"uid"`
	BinaryName      string   `yaml:"binary_name"`
	Entrypoint      []string `yaml:"entrypoint"`
	HomeDir         string   `yaml:"home_dir"`
	GasPrices       string   `yaml:"gas_prices"`
}

func ParseWorkerConfig(path string) (WorkerConfig, error) {
	file, err := os.ReadFile(path)

	if err != nil {
		return WorkerConfig{}, fmt.Errorf("failed to read config: %w", err)
	}

	var config WorkerConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return WorkerConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.DigitalOcean.Token = os.Getenv("DIGITALOCEAN_TOKEN")

	config.Tailscale.NodeAuthKey = os.Getenv("TS_NODE_AUTH_KEY")
	config.Tailscale.ServerOauthSecret = os.Getenv("TS_SERVER_OAUTH_SECRET")

	return config, nil
}

func ParseServerConfig(path string) (ServerConfig, error) {
	file, err := os.ReadFile(path)

	if err != nil {
		return ServerConfig{}, fmt.Errorf("failed to read config: %w", err)
	}

	var config ServerConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return ServerConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return config, nil
}
