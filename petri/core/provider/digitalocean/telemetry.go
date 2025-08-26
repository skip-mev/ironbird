package digitalocean

import (
	"encoding/json"
	"fmt"
)

type PrometheusSettings struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type LokiSettings struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
}

type PyroscopeSettings struct {
	URL string `json:"url"`
}

type TelemetrySettings struct {
	Prometheus PrometheusSettings `json:"prometheus"`
	Loki       LokiSettings       `json:"loki"`
	Pyroscope  PyroscopeSettings  `json:"pyroscope"`
}

type TelemetryConfig struct {
	Prometheus PrometheusSettings `json:"prometheus"`
	Loki       LokiSettings       `json:"loki"`
	Pyroscope  PyroscopeSettings  `json:"pyroscope"`
	Provider   string             `json:"provider"`
	NodeName   string             `json:"node_name"`
}

func (t *TelemetrySettings) GetCommand(provider, nodeName string) ([]string, error) {
	telemetryConfig := TelemetryConfig{
		Prometheus: t.Prometheus,
		Loki:       t.Loki,
		Pyroscope:  t.Pyroscope,
		Provider:   provider,
		NodeName:   nodeName,
	}

	config, err := json.Marshal(telemetryConfig)

	if err != nil {
		return nil, err
	}

	command := fmt.Sprintf("echo '%s' > /etc/alloy/telemetry.json", string(config))
	return []string{
		command,
		fmt.Sprintf("echo 'NODE_NAME=%s' >> /etc/environment", nodeName),
		"systemctl restart alloy",
	}, nil
}
