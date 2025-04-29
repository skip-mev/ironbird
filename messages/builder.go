package messages

type BuildDockerImageRequest struct {
	Tag            string
	Files          map[string][]byte
	BuildArguments map[string]string
}

type BuildDockerImageResponse struct {
	FQDNTag string
	Logs    []byte
}
