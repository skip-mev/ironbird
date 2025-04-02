package testnet

type RunnerType string

const (
	DigitalOcean RunnerType = "DigitalOcean"
	Docker       RunnerType = "Docker"
)

type Node struct {
	Name    string
	Rpc     string
	Lcd     string
	Metrics string
}
