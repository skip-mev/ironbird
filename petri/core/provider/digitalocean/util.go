package digitalocean

import (
	"fmt"
)

func convertEnvMapToList(env map[string]string) []string {
	envList := []string{}

	for key, value := range env {
		envList = append(envList, key+"="+value)
	}

	return envList
}

func formatUserData(commands []string) string {
	userData := `#cloud-config
runcmd:`

	for _, command := range commands {
		userData += fmt.Sprintf("\n- %s", command)
	}

	return userData
}
