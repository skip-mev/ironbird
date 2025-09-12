package messages

type BuildDockerImageRequest struct {
	Repo        string
	SHA         string
	ImageConfig ImageConfig
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
