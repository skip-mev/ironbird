package messages

type LaunchObservabilityStackRequest struct {
	ProviderState          []byte
	ProviderSpecificConfig map[string]string
	PrometheusTargets      []string
	RunnerType             string
}

type LaunchObservabilityStackResponse struct {
	ExternalGrafanaURL string
	GrafanaURL         string
	PrometheusState    []byte
	GrafanaState       []byte
	ProviderState      []byte
}
