package messages

type BuildDockerImageRequest struct {
	Repo        string
	SHA         string
	ChainConfig ChainConfig
}

type ChainConfig struct {
	Name    string
	Image   string
	Version string
}

type BuildDockerImageResponse struct {
	FQDNTag string
	Logs    []byte
}
