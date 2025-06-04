package types

import (
	"fmt"
	"time"
)

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
