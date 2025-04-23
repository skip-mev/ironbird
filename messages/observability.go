package messages

type LaunchPrometheusRequest struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusTargets      []string
	RunnerType             string
}

type LaunchPrometheusResponse struct {
	PrometheusURL   string
	PrometheusState []byte
	ProviderState   []byte
}

type LaunchGrafanaRequest struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusURL          string
	RunnerType             string
}

type LaunchGrafanaResponse struct {
	ExternalGrafanaURL string
	GrafanaURL         string
	GrafanaState       []byte
	ProviderState      []byte
}
