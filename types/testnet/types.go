package testnet

type RunnerType string

// TODO: move this out into messages
const (
	DigitalOcean RunnerType = "DigitalOcean"
	Docker       RunnerType = "Docker"
)

type Node struct {
	Name    string
	Address string
	Rpc     string
	Lcd     string
}
