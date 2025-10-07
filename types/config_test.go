package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseWorkerConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "worker_config.yaml")

	validConfigYaml := `
temporal:
  host: localhost:7233
  namespace: test-namespace
tailscale:
  server_tags:
    - test-tag
  node_tags:
    - node-tag
digitalocean:
  token: dummy-token
load_balancer:
  root_domain: test.example.com
  ssl_key_path: /path/to/ssl.key
  ssl_cert_path: /path/to/ssl.crt
telemetry:
  prometheus:
    username: prom-user
    password: prom-pass
    url: http://prometheus:9091
  loki:
    username: loki-user
    password: loki-pass
    url: http://loki:3100
builder:
  build_kit_address: tcp://buildkit:1234
  local:
    image_name: test-local-image
  ecr:
    url: test.registry.com
    image_name: test/image
  auth_env_configs:
    TEST_ENV: test_value
chains:
  test-chain:
    name: test-chain
    dockerfile: test.dockerfile
    gid: "1000"
    uid: "1000"
    binary_name: test-binary
    home_dir: /home/test
    gas_prices: 0.025stake
grafana:
  url: http://grafana:3000
  dashboards:
    - id: dashboard1
      name: test-dashboard
      human_name: Test Dashboard
server_address: localhost:9006
`
	require.NoError(t, os.WriteFile(configPath, []byte(validConfigYaml), 0644))

	t.Run("valid config", func(t *testing.T) {
		t.Setenv("DIGITALOCEAN_TOKEN", "do-token-from-env")
		t.Setenv("TS_NODE_AUTH_KEY", "tailscale-node-key")
		t.Setenv("TS_SERVER_OAUTH_SECRET", "tailscale-oauth-secret")

		config, err := ParseWorkerConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "localhost:7233", config.Temporal.Host)
		assert.Equal(t, "test-namespace", config.Temporal.Namespace)
		assert.Equal(t, []string{"test-tag"}, config.Tailscale.ServerTags)
		assert.Equal(t, []string{"node-tag"}, config.Tailscale.NodeTags)
		assert.Equal(t, "do-token-from-env", config.DigitalOcean.Token)
		assert.Equal(t, "tailscale-node-key", config.Tailscale.NodeAuthKey)
		assert.Equal(t, "tailscale-oauth-secret", config.Tailscale.ServerOauthSecret)

		assert.Equal(t, "test.example.com", config.LoadBalancer.RootDomain)
		assert.Equal(t, "/path/to/ssl.key", config.LoadBalancer.SSLKeyPath)
		assert.Equal(t, "/path/to/ssl.crt", config.LoadBalancer.SSLCertPath)

		assert.Equal(t, "prom-user", config.Telemetry.Prometheus.Username)
		assert.Equal(t, "prom-pass", config.Telemetry.Prometheus.Password)
		assert.Equal(t, "http://prometheus:9091", config.Telemetry.Prometheus.URL)
		assert.Equal(t, "loki-user", config.Telemetry.Loki.Username)
		assert.Equal(t, "loki-pass", config.Telemetry.Loki.Password)
		assert.Equal(t, "http://loki:3100", config.Telemetry.Loki.URL)

		assert.Equal(t, "tcp://buildkit:1234", config.Builder.BuildKitAddress)
		assert.Equal(t, "test-local-image", config.Builder.Local.ImageName)
		assert.Equal(t, "test.registry.com", config.Builder.ECR.URL)
		assert.Equal(t, "test/image", config.Builder.ECR.ImageName)
		assert.Equal(t, "test_value", config.Builder.AuthEnvConfigs["TEST_ENV"])

		assert.Contains(t, config.Chains, "test-chain")
		assert.Equal(t, "test-chain", config.Chains["test-chain"].Name)
		assert.Equal(t, "test.dockerfile", config.Chains["test-chain"].Dockerfile)
		assert.Equal(t, "1000", config.Chains["test-chain"].GID)
		assert.Equal(t, "1000", config.Chains["test-chain"].UID)
		assert.Equal(t, "test-binary", config.Chains["test-chain"].BinaryName)
		assert.Equal(t, "/home/test", config.Chains["test-chain"].HomeDir)
		assert.Equal(t, "0.025stake", config.Chains["test-chain"].GasPrices)

		assert.Equal(t, "http://grafana:3000", config.Grafana.URL)
		assert.Len(t, config.Grafana.Dashboards, 1)
		assert.Equal(t, "dashboard1", config.Grafana.Dashboards[0].ID)
		assert.Equal(t, "test-dashboard", config.Grafana.Dashboards[0].Name)
		assert.Equal(t, "Test Dashboard", config.Grafana.Dashboards[0].HumanName)

		assert.Equal(t, "localhost:9006", config.ServerAddress)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseWorkerConfig("non_existent_file.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid.yaml")
		require.NoError(t, os.WriteFile(invalidPath, []byte("invalid: yaml: content"), 0644))

		_, err := ParseWorkerConfig(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})

}

func TestParseServerConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "server_config.yaml")

	validConfigYaml := `
temporal:
  host: localhost:7233
  namespace: server-namespace
database_path: ./test.db
migrations_path: ./test-migrations
grpc_address: localhost:9007
grpc_web_address: localhost:9008
`
	require.NoError(t, os.WriteFile(configPath, []byte(validConfigYaml), 0644))

	t.Run("valid config", func(t *testing.T) {
		config, err := ParseServerConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "localhost:7233", config.Temporal.Host)
		assert.Equal(t, "server-namespace", config.Temporal.Namespace)
		assert.Equal(t, "./test.db", config.DatabasePath)
		assert.Equal(t, "./test-migrations", config.MigrationsPath)
		assert.Equal(t, "localhost:9007", config.GrpcAddress)
		assert.Equal(t, "localhost:9008", config.GrpcWebAddress)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseServerConfig("non_existent_file.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid_server.yaml")
		require.NoError(t, os.WriteFile(invalidPath, []byte("invalid: yaml: content"), 0644))

		_, err := ParseServerConfig(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})

}
