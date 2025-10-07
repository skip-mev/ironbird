package messages

type BuildDockerImageRequest struct {
	Repo         string
	SHA          string
	ImageConfig  ImageConfig
	CosmosSdkSha string // Optional: SHA/version to replace cosmos-sdk dependency
	CometBFTSha  string // Optional: SHA/version to replace cometbft dependency
}

type ImageConfig struct {
	Name    string
	Image   string
	Version string
}

type BuildDockerImageResponse struct {
	FQDNTag string
	Logs    []byte
}
