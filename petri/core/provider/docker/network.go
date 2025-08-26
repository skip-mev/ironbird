package docker

import (
	"context"
	"fmt"
	"math/rand"
	"net"

	"github.com/docker/docker/api/types/network"
	"go.uber.org/zap"

	"github.com/docker/go-connections/nat"
)

const (
	providerLabelName = "petri-provider"
	portsLabelName    = "petri-ports"
	nodeNameLabelName = "petri-node-name"
)

func (p *Provider) initNetwork(ctx context.Context) (network.Inspect, error) {
	state := p.GetState()

	p.logger.Info("creating network", zap.String("name", state.NetworkName))
	subnet1 := rand.Intn(255)
	subnet2 := rand.Intn(255)

	networkResponse, err := p.dockerClient.NetworkCreate(ctx, state.NetworkName, network.CreateOptions{
		Scope:  "local",
		Driver: "bridge",
		Options: map[string]string{ // https://docs.docker.com/engine/reference/commandline/network_create/#bridge-driver-options
			"com.docker.network.bridge.enable_icc":           "true",
			"com.docker.network.bridge.enable_ip_masquerade": "true",
			"com.docker.network.bridge.host_binding_ipv4":    "0.0.0.0",
			"com.docker.network.driver.mtu":                  "1500",
		},
		Internal:   false,
		Attachable: false,
		Ingress:    false,
		Labels: map[string]string{
			providerLabelName: state.Name,
		},
		IPAM: &network.IPAM{
			Driver: "default",
			Config: []network.IPAMConfig{
				{
					Subnet:  fmt.Sprintf("192.%d.%d.0/24", subnet1, subnet2),
					Gateway: fmt.Sprintf("192.%d.%d.1", subnet1, subnet2),
				},
			},
		},
	})

	if err != nil {
		return network.Inspect{}, err
	}

	networkInfo, err := p.dockerClient.NetworkInspect(ctx, networkResponse.ID, network.InspectOptions{})
	if err != nil {
		return network.Inspect{}, err
	}

	return networkInfo, nil
}

// ensureNetwork checks if the network exists and has the expected configuration.
func (p *Provider) ensureNetwork(ctx context.Context) error {
	state := p.GetState()
	network, err := p.dockerClient.NetworkInspect(ctx, state.NetworkID, network.InspectOptions{})

	if err != nil {
		return err
	}

	if network.ID != state.NetworkID {
		return fmt.Errorf("network ID mismatch: %s != %s", network.ID, state.NetworkID)
	}

	if network.Name != state.NetworkName {
		return fmt.Errorf("network name mismatch: %s != %s", network.Name, state.NetworkName)
	}

	if len(network.IPAM.Config) != 1 {
		return fmt.Errorf("unexpected number of IPAM configs: %d", len(network.IPAM.Config))
	}

	if network.IPAM.Config[0].Subnet != state.NetworkCIDR {
		return fmt.Errorf("network CIDR mismatch: %s != %s", network.IPAM.Config[0].Subnet, state.NetworkCIDR)
	}

	return nil
}

func (p *Provider) destroyNetwork(ctx context.Context) error {
	return p.dockerClient.NetworkRemove(ctx, p.GetState().NetworkID)
}

// openListenerOnFreePort opens the next free port
func (p *Provider) openListenerOnFreePort() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// nextAvailablePort generates a docker PortBinding by finding the next available port.
// The listener will be closed in the case of an error, otherwise it will be left open.
// This allows multiple nextAvailablePort calls to find multiple available ports
// before closing them so they are available for the PortBinding.
func (p *Provider) nextAvailablePort() (nat.PortBinding, *net.TCPListener, error) {
	// TODO: add listeners to state
	p.networkMu.Lock()
	defer p.networkMu.Unlock()

	l, err := p.openListenerOnFreePort()
	if err != nil {
		if l != nil {
			if err := l.Close(); err != nil {
				return nat.PortBinding{}, nil, err
			}
		}

		return nat.PortBinding{}, nil, err
	}

	return nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: fmt.Sprint(l.Addr().(*net.TCPAddr).Port),
	}, l, nil
}

func (p *Provider) nextAvailableIP() (string, error) {
	p.networkMu.Lock()
	p.stateMu.Lock()
	defer p.networkMu.Unlock()
	defer p.stateMu.Unlock()

	ip, err := p.dockerNetworkAllocator.AllocateNext()

	if err != nil {
		return "", err
	}

	return ip.String(), nil
}

// GeneratePortBindings will find open ports on the local
// machine and create a PortBinding for every port in the portSet.
func (p *Provider) GeneratePortBindings(portSet nat.PortSet) (nat.PortMap, error) {
	m := make(nat.PortMap)

	for port := range portSet {
		pb, l, err := p.nextAvailablePort()

		if err != nil {
			return nat.PortMap{}, err
		}

		err = l.Close()

		if err != nil {
			return nat.PortMap{}, err
		}

		m[port] = []nat.PortBinding{pb}
	}

	return m, nil
}
