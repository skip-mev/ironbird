package docker_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/skip-mev/ironbird/petri/core/provider/clients"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"
	"go.uber.org/zap/zaptest"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const idAlphabet = "abcdefghijklqmnoqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func setupTest(tb testing.TB, name string) func(testing.TB, string) {
	return func(tb testing.TB, name string) {
		client, _ := client.NewClientWithOpts()

		nets, _ := client.NetworkList(context.Background(), network.ListOptions{
			Filters: filters.NewArgs(filters.Arg("name", fmt.Sprintf("petri-network-%s", name))),
		})

		if len(nets) != 0 {
			for _, net := range nets {
				err := client.NetworkRemove(context.Background(), net.ID)
				require.NoError(tb, err)
			}
		}
	}
}

func TestCreateProviderDuplicateNetwork(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	teardown := setupTest(t, providerName)
	defer teardown(t, providerName)

	p1, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)
	defer func(ctx context.Context, p provider.ProviderI) {
		require.NoError(t, p.Teardown(ctx))
	}(ctx, p1)

	p2, err := docker.CreateProvider(ctx, logger, providerName)
	require.Error(t, err)
	require.Nil(t, p2)
}

func TestCreateProvider(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	teardown := setupTest(t, providerName)
	defer teardown(t, providerName)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)
	defer func(ctx context.Context, p provider.ProviderI) {
		require.NoError(t, p.Teardown(ctx))
	}(ctx, p)

	state := p.GetState()
	assert.Equal(t, providerName, state.Name)
	assert.NotEmpty(t, state.NetworkID)
	assert.NotEmpty(t, state.NetworkName)
	assert.NotEmpty(t, state.NetworkCIDR)
}

func TestRestoreProvider(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	teardown := setupTest(t, providerName)
	defer teardown(t, providerName)

	p1, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	state1 := p1.GetState()
	serialized, err := p1.SerializeProvider(ctx)
	require.NoError(t, err)

	p2, err := docker.RestoreProvider(ctx, logger, serialized)
	require.NoError(t, err)

	state2 := p2.GetState()
	assert.Equal(t, state1, state2)
}

func TestCreateTask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)
	ctx := context.Background()

	p, err := docker.CreateProvider(context.Background(), logger, providerName)
	require.NoError(t, err)

	defer func(ctx context.Context, p provider.ProviderI) {
		require.NoError(t, p.Teardown(ctx))
	}(ctx, p)

	tests := []struct {
		name       string
		definition provider.TaskDefinition
		wantErr    bool
		errorMsg   string
	}{
		{
			name: "success",
			definition: provider.TaskDefinition{
				Name:  fmt.Sprintf("%s-test-task-success", providerName),
				Image: provider.ImageDefinition{Image: "busybox:latest", GID: "1000", UID: "1000"},
			},
			wantErr: false,
		},
		{
			name: "missing image",
			definition: provider.TaskDefinition{
				Name: fmt.Sprintf("%s-test-task-no-image", providerName),
			},
			wantErr:  true,
			errorMsg: "image cannot be empty",
		},
		{
			name: "invalid image",
			definition: provider.TaskDefinition{
				Name:  fmt.Sprintf("%s-test-task-bad-image", providerName),
				Image: provider.ImageDefinition{Image: "busybox:latest"},
			},
			wantErr:  true,
			errorMsg: "image definition is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			task, err := p.CreateTask(context.Background(), tt.definition)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				return
			}

			assert.NoError(t, err)

			dockerTask, ok := task.(*docker.Task)
			assert.True(t, ok)
			assert.NotNil(t, dockerTask)

			state := dockerTask.GetState()
			assert.NotEmpty(t, state.Id)
			assert.Equal(t, fmt.Sprintf("%s-%s", providerName, tt.definition.Name), state.Name)
		})
	}
}

func TestConcurrentTaskCreation(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	defer func(ctx context.Context, p provider.ProviderI) {
		require.NoError(t, p.Teardown(ctx))
	}(ctx, p)

	numTasks := 10
	var wg sync.WaitGroup
	errors := make(chan error, numTasks)
	tasks := make(chan *docker.Task, numTasks)

	for i := 0; i < numTasks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			task, err := p.CreateTask(ctx, provider.TaskDefinition{
				Name:       fmt.Sprintf("task-%s-%d", providerName, index),
				Image:      provider.ImageDefinition{Image: "busybox:latest", GID: "1000", UID: "1000"},
				Entrypoint: []string{"sh", "-c"},
				Command:    []string{"sleep 1000"},
			})
			if err != nil {
				errors <- err
				return
			}

			err = task.Start(ctx)
			if err != nil {
				errors <- err
				return
			}
			tasks <- task.(*docker.Task)
		}(i)
	}

	wg.Wait()
	close(errors)
	close(tasks)

	for err := range errors {
		assert.NoError(t, err)
	}

	providerState := p.GetState()
	ips := make(map[string]bool)

	for task := range tasks {
		taskState := task.GetState()
		dockerClient, _ := clients.NewDockerClient("", nil)
		containerJSON, err := dockerClient.ContainerInspect(ctx, taskState.Id)
		require.NoError(t, err)

		ip := containerJSON.NetworkSettings.Networks[providerState.NetworkName].IPAddress
		assert.False(t, ips[ip], "Duplicate IP found: %s", ip)
		ips[ip] = true
	}
}

func TestProviderSerialization(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)

	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	p1, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	defer func(ctx context.Context, p provider.ProviderI) {
		require.NoError(t, p.Teardown(ctx))
	}(ctx, p1)

	_, err = p1.CreateTask(ctx, provider.TaskDefinition{
		Name:  fmt.Sprintf("%s-test-task", providerName),
		Image: provider.ImageDefinition{Image: "busybox:latest", GID: "1000", UID: "1000"},
	})
	require.NoError(t, err)

	state1 := p1.GetState()
	serialized, err := p1.SerializeProvider(ctx)
	require.NoError(t, err)

	p2, err := docker.RestoreProvider(ctx, logger, serialized)
	require.NoError(t, err)

	state2 := p2.GetState()
	assert.Equal(t, state1, state2)
}

func TestTeardown(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)
	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name:    fmt.Sprintf("%s-test-task-teardown", providerName),
		Image:   provider.ImageDefinition{Image: "busybox:latest", GID: "1000", UID: "1000"},
		DataDir: "/data",
	})
	require.NoError(t, err)

	dockerTask := task.(*docker.Task)
	taskState := dockerTask.GetState()
	err = p.Teardown(ctx)
	assert.NoError(t, err)

	client, _ := client.NewClientWithOpts()
	_, err = client.ContainerInspect(ctx, taskState.Id)
	assert.Error(t, err)
}

func TestRestoreTask(t *testing.T) {
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	providerName := gonanoid.MustGenerate(idAlphabet, 10)

	teardown := setupTest(t, providerName)
	defer teardown(t, providerName)

	p, err := docker.CreateProvider(ctx, logger, providerName)
	require.NoError(t, err)

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name:    fmt.Sprintf("%s-test-task-restore", providerName),
		Image:   provider.ImageDefinition{Image: "busybox:latest", GID: "1000", UID: "1000"},
		DataDir: "/data",
	})
	require.NoError(t, err)

	dockerTask := task.(*docker.Task)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)

	restoredTask, err := p.DeserializeTask(ctx, state)
	require.NoError(t, err)

	restoredDockerTask, ok := restoredTask.(*docker.Task)
	require.True(t, ok)
	require.NotNil(t, restoredDockerTask)
	require.Equal(t, dockerTask.GetState(), restoredDockerTask.GetState())
}
