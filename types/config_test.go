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
telemetry:
  prometheus:
    username: prom-user
    password: prom-pass
    url: http://prometheus:9090
  loki:
    username: loki-user
    password: loki-pass
    url: http://loki:3100
builder:
  build_kit_address: tcp://buildkit:1234
  registry:
    url: test.registry.com
    image_name: test/image
  auth_env_configs:
    TEST_ENV: test_value
github:
  app:
    integration_id: 12345
    private_key: dGVzdC1wcml2YXRlLWtleQ== # base64 encoded "test-private-key"
    webhook_secret: webhook-secret
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
		assert.Equal(t, "test-private-key", config.Github.App.PrivateKey)
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

func TestParseAppConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "app_config.yaml")

	validConfigYaml := `
temporal:
  host: localhost:7233
  namespace: app-namespace
grafana:
  url: http://grafana:3000
  dashboards:
    - id: dashboard1
      name: test-dashboard
      human_name: Test Dashboard
chains:
  test-chain:
    name: test-chain
    version: v1.0.0
    num_of_nodes: 4
    num_of_validators: 3
    image:
      dockerfile: Dockerfile
      gid: "1000"
      uid: "1000"
      binary_name: test-binary
      home_dir: /home/app
      gas_prices: 0.0test
github:
  app:
    integration_id: 12345
    private_key: dGVzdC1wcml2YXRlLWtleQ== # base64 encoded "test-private-key"
    webhook_secret: webhook-secret
`
	require.NoError(t, os.WriteFile(configPath, []byte(validConfigYaml), 0644))

	t.Run("valid config", func(t *testing.T) {
		config, err := ParseAppConfig(configPath)
		require.NoError(t, err)

		assert.Equal(t, "localhost:7233", config.Temporal.Host)
		assert.Equal(t, "app-namespace", config.Temporal.Namespace)
		assert.Equal(t, "http://grafana:3000", config.Grafana.URL)
		assert.Len(t, config.Grafana.Dashboards, 1)
		assert.Equal(t, "dashboard1", config.Grafana.Dashboards[0].ID)
		assert.Equal(t, "test-dashboard", config.Grafana.Dashboards[0].Name)
		assert.Equal(t, "Test Dashboard", config.Grafana.Dashboards[0].HumanName)

		assert.Contains(t, config.Chains, "test-chain")
		assert.Equal(t, "test-chain", config.Chains["test-chain"].Name)
		assert.Equal(t, uint64(4), config.Chains["test-chain"].NumOfNodes)
		assert.Equal(t, "test-private-key", config.Github.App.PrivateKey)
	})

	t.Run("file not found", func(t *testing.T) {
		_, err := ParseAppConfig("non_existent_file.yaml")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "invalid_app.yaml")
		require.NoError(t, os.WriteFile(invalidPath, []byte("invalid: yaml: content"), 0644))

		_, err := ParseAppConfig(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal config")
	})

}
