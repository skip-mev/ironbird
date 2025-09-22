package e2e

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/types"
	cosmoschain "github.com/skip-mev/ironbird/petri/cosmos/chain"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func AssertNodeRunning(t *testing.T, ctx context.Context, node types.NodeI) {
	status, err := node.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, status)

	ip, err := node.GetIP(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, ip)

	testFile := "test.txt"
	testContent := []byte("test content")
	err = node.WriteFile(ctx, testFile, testContent)
	require.NoError(t, err)

	readContent, err := node.ReadFile(ctx, testFile)
	require.NoError(t, err)
	require.Equal(t, testContent, readContent)
}

func AssertNodeShutdown(t *testing.T, ctx context.Context, node types.NodeI) {
	status, err := node.GetStatus(ctx)
	require.Error(t, err)
	require.Equal(t, provider.TASK_STATUS_UNDEFINED, status, "node status should report as undefined after shutdown")
}

func GetExternalIP() (string, error) {
	resp, err := http.Get("https://ifconfig.me")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ip, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(ip)), nil
}

// CreateChainsConcurrently creates multiple chains concurrently using the provided configuration
func CreateChainsConcurrently(
	ctx context.Context,
	t *testing.T,
	logger *zap.Logger,
	p provider.ProviderI,
	startIndex, endIndex int,
	chains []*cosmoschain.Chain,
	chainConfig types.ChainConfig,
	chainIDFmtStr string,
	chainOptions types.ChainOptions,
) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for i := startIndex; i < endIndex; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Create a copy of the config to avoid race conditions
			config := chainConfig
			config.ChainId = fmt.Sprintf(chainIDFmtStr, index+1)
			config.Name = fmt.Sprintf("chain-%d", index+1)

			c, err := cosmoschain.CreateChain(ctx, logger, p, config, chainOptions)
			if err != nil {
				t.Logf("Chain creation error for chain %d: %v", index+1, err)
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to create chain %d: %w", index+1, err))
				mu.Unlock()
				return
			}

			if err := c.Init(ctx, chainOptions); err != nil {
				t.Logf("Chain initialization error for chain %d: %v", index+1, err)
				mu.Lock()
				errors = append(errors, fmt.Errorf("failed to init chain %d: %w", index+1, err))
				mu.Unlock()
				return
			}

			// Safely write to the chains slice
			mu.Lock()
			chains[index] = c
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Check for any errors that occurred during chain creation
	if len(errors) > 0 {
		for _, err := range errors {
			t.Errorf("Chain creation failed: %v", err)
		}
		require.Empty(t, errors, "Chain creation failed with %d errors", len(errors))
	}
}
