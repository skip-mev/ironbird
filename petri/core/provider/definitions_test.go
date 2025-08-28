package provider_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/skip-mev/ironbird/petri/core/provider"
)

var validImageDefinition = provider.ImageDefinition{
	Image: "test",
	GID:   "1000",
	UID:   "1000",
}

func TestImageDefinitionValidation(t *testing.T) {
	tcs := []struct {
		name       string
		def        provider.ImageDefinition
		expectPass bool
	}{
		{
			name: "valid",
			def: provider.ImageDefinition{
				Image: "test",
				GID:   "1000",
				UID:   "1000",
			},
			expectPass: true,
		},
		{
			name: "empty image",
			def: provider.ImageDefinition{
				Image: "",
				GID:   "1000",
				UID:   "1000",
			},
		},
		{
			name: "empty uid",
			def: provider.ImageDefinition{
				Image: "test",
				GID:   "1000",
				UID:   "",
			},
			expectPass: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.def.ValidateBasic()
			if tc.expectPass {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestVolumeDefinitionValidation(t *testing.T) {
	tcs := []struct {
		name       string
		def        provider.VolumeDefinition
		expectPass bool
	}{
		{
			name: "valid",
			def: provider.VolumeDefinition{
				MountPath: "/tmp",
				Name:      "test",
				Size:      "100",
			},
			expectPass: true,
		},
		{
			name: "empty mountpath",
			def: provider.VolumeDefinition{
				MountPath: "",
				Name:      "test",
				Size:      "100",
			},
			expectPass: false,
		},
		{
			name: "empty name",
			def: provider.VolumeDefinition{
				MountPath: "/tmp",
				Name:      "",
				Size:      "100",
			},
			expectPass: false,
		},
		{
			name: "empty size",
			def: provider.VolumeDefinition{
				MountPath: "/tmp",
				Name:      "test",
				Size:      "",
			},
			expectPass: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.def.ValidateBasic()
			if tc.expectPass {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}

func TestTaskDefinitionValidation(t *testing.T) {
	tcs := []struct {
		name       string
		def        provider.TaskDefinition
		expectPass bool
	}{
		{
			name: "valid",
			def: provider.TaskDefinition{
				Name:  "test",
				Image: validImageDefinition,
			},
			expectPass: true,
		},
		{
			name: "no name",
			def: provider.TaskDefinition{
				Image: validImageDefinition,
			},
			expectPass: false,
		},
		{
			name: "no image",
			def: provider.TaskDefinition{
				Name: "test",
			},
			expectPass: false,
		},
		{
			name: "invalid image",
			def: provider.TaskDefinition{
				Name: "test",
				Image: provider.ImageDefinition{
					Image: "",
				},
			},
			expectPass: false,
		},
		{
			name: "invalid port",
			def: provider.TaskDefinition{
				Name:  "test",
				Image: validImageDefinition,
				Ports: []string{"", "100000"},
			},
			expectPass: false,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.def.ValidateBasic()
			if tc.expectPass {
				assert.Nil(t, err)
			} else {
				assert.NotNil(t, err)
			}
		})
	}
}
