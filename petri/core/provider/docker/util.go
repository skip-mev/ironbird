package docker

import (
	"fmt"

	"github.com/docker/go-connections/nat"

	"github.com/skip-mev/ironbird/petri/core/provider"
)

func convertTaskDefinitionPortsToPortSet(definition provider.TaskDefinition) nat.PortSet {
	bindings := nat.PortSet{}

	for _, port := range definition.Ports {
		bindings[nat.Port(fmt.Sprintf("%s/tcp", port))] = struct{}{}
	}

	return bindings
}

func convertEnvMapToList(env map[string]string) []string {
	envList := []string{}

	for key, value := range env {
		envList = append(envList, key+"="+value)
	}

	return envList
}
