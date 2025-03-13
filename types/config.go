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

type AppConfig struct {
	Github   githubapp.Config
	Chains   []ChainsConfig `yaml:"chains"`
	Temporal TemporalConfig `yaml:"temporal"`
}

type WorkerConfig struct {
	Temporal     TemporalConfig     `yaml:"temporal"`
	Tailscale    TailscaleConfig    `yaml:"tailscale"`
	DigitalOcean DigitalOceanConfig `yaml:"digitalocean"`
	Builder      BuilderConfig      `yaml:"builder"`
	Github       githubapp.Config
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
	Username string `yaml:"username"`
	Password string
	URL      string `yaml:"url"`
	FQDN     string `yaml:"fqdn"`
}

type ChainsConfig struct {
	Name                 string                 `yaml:"name"`
	SnapshotURL          string                 `yaml:"snapshot_url"`
	Dependencies         map[string]string      `yaml:"dependencies"`
	Image                ImageConfig            `yaml:"image"`
	Version              string                 `yaml:"version"`
	GenesisModifications []petrichain.GenesisKV `yaml:"genesis_modifications"`
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

	config.Builder.Registry.Password = os.Getenv("REGISTRY_TOKEN")

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

func ParseAppConfig(path string) (AppConfig, error) {
	file, err := os.ReadFile(path)

	if err != nil {
		return AppConfig{}, fmt.Errorf("failed to read config: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(file, &config); err != nil {
		return AppConfig{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	config.Github.SetValuesFromEnv("")

	if decodedGithubKey, err := base64.StdEncoding.DecodeString(config.Github.App.PrivateKey); err == nil {
		config.Github.App.PrivateKey = string(decodedGithubKey)
	} else {
		return AppConfig{}, err
	}

	return config, nil
}
