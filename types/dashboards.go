package types

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

type DashboardsConfig struct {
	Grafana GrafanaConfig `yaml:"grafana"`
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

func ParseDashboardsConfig(filePath string) (*DashboardsConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read dashboards config file: %w", err)
	}

	var config DashboardsConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse dashboards config: %w", err)
	}

	return &config, nil
}

func (c *DashboardsConfig) GenerateMonitoringLinks(chainID string, startTime time.Time) map[string]string {
	urls := make(map[string]string)

	for _, dashboard := range c.Grafana.Dashboards {
		url := fmt.Sprintf("%s/d/%s/%s?orgId=1&var-chain_id=%s&from=%d&to=%s&refresh=auto",
			c.Grafana.URL, dashboard.ID, dashboard.Name, chainID, startTime.UnixMilli(), "now")
		urls[dashboard.HumanName] = url
	}

	return urls
}
