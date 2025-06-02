package messages

import (
	"testing"

	"github.com/skip-mev/ironbird/types"
	"github.com/skip-mev/ironbird/types/testnet"
	"github.com/stretchr/testify/assert"
)

func TestTestnetWorkflowRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request TestnetWorkflowRequest
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid request",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: false,
		},
		{
			name: "missing installation ID",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "installationID is required",
		},
		{
			name: "missing repo",
			request: TestnetWorkflowRequest{
				Repo: "",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "repo is required",
		},
		{
			name: "missing SHA",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "SHA is required",
		},
		{
			name: "missing chain name",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "chain name is required",
		},
		{
			name: "missing binary name",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "binary name is required",
		},
		{
			name: "missing home dir",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "",
					},
				},
				RunnerType: testnet.Docker,
			},
			wantErr: true,
			errMsg:  "home directory is required",
		},
		{
			name: "invalid runner type",
			request: TestnetWorkflowRequest{
				Repo: "ironbird",
				SHA:  "abcdef123456",
				ChainConfig: types.ChainsConfig{
					Name: "test-chain",
					Image: types.ImageConfig{
						BinaryName: "test-binary",
						HomeDir:    "/home/test",
					},
				},
				RunnerType: "invalid-runner",
			},
			wantErr: true,
			errMsg:  "runner type must be one of: DigitalOcean, Docker",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
