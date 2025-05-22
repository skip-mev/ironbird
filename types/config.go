package types

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/palantir/go-githubapp/githubapp"
	petrichain "github.com/skip-mev/petri/cosmos/v3/chain"
	"gopkg.in/yaml.v3"
)

type TailscaleConfig struct {
	ServerOauthSecret string
	ServerTags        []string `yaml:"server_tags"`
	NodeAuthKey       string
	NodeTags          []string `yaml:"node_tags"`
}

type WorkerConfig struct {
	Temporal     TemporalConfig     `yaml:"temporal"`
	Tailscale    TailscaleConfig    `yaml:"tailscale"`
	DigitalOcean DigitalOceanConfig `yaml:"digitalocean"`
	LoadBalancer LoadBalancerConfig `yaml:"load_balancer"`
	Telemetry    TelemetryConfig    `yaml:"telemetry"`
	Builder      BuilderConfig      `yaml:"builder"`
	Github       githubapp.Config
}

type LoadBalancerConfig struct {
	RootDomain  string `yaml:"root_domain"`
	SSLKeyPath  string `yaml:"ssl_key_path"`
	SSLCertPath string `yaml:"ssl_cert_path"`
}

type TelemetryConfig struct {
	Prometheus PrometheusConfig `yaml:"prometheus"`
	Loki       LokiConfig       `yaml:"loki"`
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

type GrafanaConfig struct {
	URL        string             `yaml:"url"`
	Dashboards []GrafanaDashboard `yaml:"dashboards"`
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
	Name                 string                 `yaml:"name"`
	Dependencies         map[string]string      `yaml:"dependencies"`
	Image                ImageConfig            `yaml:"image"`
	Version              string                 `yaml:"version"`
	GenesisModifications []petrichain.GenesisKV `yaml:"genesis_modifications"`
	NumOfNodes           uint64                 `yaml:"num_of_nodes"`
	NumOfValidators      uint64                 `yaml:"num_of_validators"`
}

type ImageConfig struct {
	Dockerfile string `yaml:"dockerfile"`
	GID        string `yaml:"gid"`
	UID        string `yaml:"uid"`
	BinaryName string `yaml:"binary_name"`
	HomeDir    string `yaml:"home_dir"`
	GasPrices  string `yaml:"gas_prices"`
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

	config.Github.SetValuesFromEnv("")
	if decodedGithubKey, err := base64.StdEncoding.DecodeString(config.Github.App.PrivateKey); err == nil {
		config.Github.App.PrivateKey = string(decodedGithubKey)
	} else {
		return WorkerConfig{}, err
	}

	config.DigitalOcean.Token = os.Getenv("DIGITALOCEAN_TOKEN")

	config.Tailscale.NodeAuthKey = os.Getenv("TS_NODE_AUTH_KEY")
	config.Tailscale.ServerOauthSecret = os.Getenv("TS_SERVER_OAUTH_SECRET")

	return config, nil
}
