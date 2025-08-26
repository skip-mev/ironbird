package provider

import (
	"fmt"
	"strconv"
)

// VolumeDefinition defines the configuration for a volume. Some providers might not support creating a volume,
// but this detail is abstracted away when CreateTask is used
type VolumeDefinition struct {
	Name      string
	MountPath string
	Size      string
}

func (v *VolumeDefinition) ValidateBasic() error {
	if v.Name == "" {
		return fmt.Errorf("volume name cannot be empty")
	}

	if v.MountPath == "" {
		return fmt.Errorf("mount path cannot be empty")
	}

	if v.Size == "" {
		return fmt.Errorf("size cannot be empty")
	}

	return nil
}

// TaskDefinition defines the configuration for a task
type TaskDefinition struct {
	Name        string // Name is used when generating volumes, etc. - additional resources for the container
	Image       ImageDefinition
	Ports       []string
	Environment map[string]string
	DataDir     string
	Entrypoint  []string
	Command     []string
	Args        []string

	ProviderSpecificConfig map[string]string
}

func (t *TaskDefinition) ValidateBasic() error {
	if t.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if err := t.Image.ValidateBasic(); err != nil {
		return fmt.Errorf("image definition is invalid: %w", err)
	}

	for _, port := range t.Ports {
		if port == "" {
			return fmt.Errorf("port cannot be empty")
		}

		portInt, err := strconv.ParseUint(port, 10, 64)
		if err != nil {
			return fmt.Errorf("port must be a valid unsigned integer")
		}

		if portInt > 65535 {
			return fmt.Errorf("port must be less than 65535")
		}
	}

	return nil
}

// ImageDefinition defines the details of a specific Docker image
type ImageDefinition struct {
	Image string
	UID   string
	GID   string
}

func (i *ImageDefinition) ValidateBasic() error {
	if i.Image == "" {
		return fmt.Errorf("image cannot be empty")
	}

	if i.UID == "" {
		return fmt.Errorf("uid cannot be empty")
	}

	if _, err := strconv.ParseUint(i.UID, 10, 64); err != nil {
		return fmt.Errorf("uid must be a valid unsigned integer")
	}

	if i.GID == "" {
		return fmt.Errorf("gid cannot be empty")
	}

	if _, err := strconv.ParseUint(i.GID, 10, 64); err != nil {
		return fmt.Errorf("gid must be a valid unsigned integer")
	}

	return nil
}
