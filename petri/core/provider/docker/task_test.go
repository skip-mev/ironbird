package docker_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/docker/docker/client"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/docker"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestTaskLifecycle(t *testing.T) {
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

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "test",
		Image: provider.ImageDefinition{
			Image: "nginx:latest",
			UID:   "1000",
			GID:   "1000",
		},
		Ports: []string{
			"80",
		},
	})
	require.NoError(t, err)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	err = task.Start(ctx)
	require.NoError(t, err)
	time.Sleep(time.Second * 5)

	err = task.Stop(ctx)
	require.NoError(t, err)

	require.Equal(t, 1, len(p.GetState().TaskStates))
	err = task.Destroy(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(p.GetState().TaskStates))

	dockerTask, ok := task.(*docker.Task)
	require.True(t, ok)

	client, _ := client.NewClientWithOpts()
	_, err = client.ContainerInspect(ctx, dockerTask.GetState().Id)

	require.Error(t, err)
}

func TestTaskExposingPort(t *testing.T) {
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

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "test",
		Image: provider.ImageDefinition{
			Image: "nginx:latest",
			UID:   "1000",
			GID:   "1000",
		},
		Ports: []string{
			"80",
		},
	})
	require.NoError(t, err)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	err = task.Start(ctx)
	require.NoError(t, err)

	ip, err := task.GetExternalAddress(ctx, "80")
	require.NoError(t, err)
	require.NotEmpty(t, ip)

	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s", ip), nil)
	require.NoError(t, err)
	require.NotEmpty(t, req)

	err = task.Destroy(ctx)
	require.NoError(t, err)
}

func TestTaskRunCommand(t *testing.T) {
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

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "test",
		Image: provider.ImageDefinition{
			Image: "busybox:latest",
			UID:   "1000",
			GID:   "1000",
		},
		Entrypoint: []string{"sh", "-c"},
		Command:    []string{"sleep 36000"},
	})
	require.NoError(t, err)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	err = task.Start(ctx)
	require.NoError(t, err)

	stdout, stderr, exitCode, err := task.RunCommand(ctx, []string{"sh", "-c", "echo hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.Equal(t, "hello\n", stdout)

	stdout, stderr, exitCode, err = task.RunCommand(ctx, []string{"sh", "-c", "echo hello >&2"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stdout)
	require.Equal(t, "hello\n", stderr)

	err = task.Destroy(ctx)
	require.NoError(t, err)
}

func TestTaskRunCommandWhileStopped(t *testing.T) {
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

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "test",
		Image: provider.ImageDefinition{
			Image: "busybox:latest",
			UID:   "1000",
			GID:   "1000",
		},
		Entrypoint: []string{"sh", "-c"},
		Command:    []string{"sleep 36000"},
	})
	require.NoError(t, err)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	stdout, stderr, exitCode, err := task.RunCommand(ctx, []string{"sh", "-c", "echo hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.Equal(t, "hello\n", stdout)

	stdout, stderr, exitCode, err = task.RunCommand(ctx, []string{"sh", "-c", "echo hello >&2"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stdout)
	require.Equal(t, "hello\n", stderr)

	err = task.Destroy(ctx)
	require.NoError(t, err)
}

func TestTaskReadWriteFile(t *testing.T) {
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

	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "test",
		Image: provider.ImageDefinition{
			Image: "busybox:latest",
			UID:   "1000",
			GID:   "1000",
		},
		Entrypoint: []string{"sh", "-c"},
		Command:    []string{"sleep 36000"},
		DataDir:    "/data",
	})
	require.NoError(t, err)

	state, err := p.SerializeTask(ctx, task)
	require.NoError(t, err)
	require.NotEmpty(t, state)

	err = task.WriteFile(ctx, "test.txt", []byte("hello world"))
	require.NoError(t, err)

	stdout, stderr, exitCode, err := task.RunCommand(ctx, []string{"sh", "-c", "cat /data/test.txt"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)
	require.Equal(t, "hello world", stdout)

	out, err := task.ReadFile(ctx, "test.txt")
	require.NoError(t, err)
	require.Equal(t, "hello world", string(out))

	err = task.Destroy(ctx)
	require.NoError(t, err)
}
