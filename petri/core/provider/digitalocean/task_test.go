package digitalocean

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	clientmocks "github.com/skip-mev/ironbird/petri/core/provider/clients/mocks"

	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockConn implements net.Conn interface for testing
type mockConn struct {
	*bytes.Buffer
}

// mockNotFoundError implements errdefs.ErrNotFound interface for testing
type mockNotFoundError struct {
	error
}

func (e mockNotFoundError) NotFound() {}

func (m mockConn) Close() error                       { return nil }
func (m mockConn) LocalAddr() net.Addr                { return nil }
func (m mockConn) RemoteAddr() net.Addr               { return nil }
func (m mockConn) SetDeadline(t time.Time) error      { return nil }
func (m mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m mockConn) SetWriteDeadline(t time.Time) error { return nil }

const (
	testContainerID = "test-container-id"
)

var (
	testContainer = types.Container{ID: testContainerID}
	testDroplet   = &godo.Droplet{ID: 123, Status: "active"}
)

func TestTaskLifecycle(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	droplet := &godo.Droplet{
		ID:     123,
		Status: "active",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{
					Type:      "public",
					IPAddress: "10.0.0.1",
				},
			},
		},
	}

	mockDO.On("GetDroplet", ctx, droplet.ID).Return(droplet, nil)

	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "running",
			},
		},
	}, nil)

	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)
	mockDocker.On("ContainerStop", ctx, testContainerID, container.StopOptions{}).Return(nil)

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(droplet.ID),
			Name:         "test-task",
			ProviderName: "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
			},
			Status: provider.TASK_STOPPED,
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	err := task.Start(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, task.GetState().Status)

	status, err := task.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, status)

	err = task.Stop(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_STOPPED, task.GetState().Status)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestTaskRunCommand(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	mockDO.On("GetDroplet", ctx, 1).Return(testDroplet, nil)

	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "running",
			},
		},
	}, nil)

	execID := "test-exec-id"
	execCreateResp := types.IDResponse{ID: execID}
	mockDocker.On("ContainerExecCreate", ctx, testContainerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"echo", "hello"},
	}).Return(execCreateResp, nil)

	conn := &mockConn{Buffer: bytes.NewBuffer([]byte{})}
	mockDocker.On("ContainerExecAttach", ctx, execID, container.ExecAttachOptions{}).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}, nil)
	mockDocker.On("ContainerExecInspect", ctx, execID).Return(container.ExecInspect{
		ExitCode: 0,
		Running:  false,
	}, nil)

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(1),
			Name:         "test-task",
			ProviderName: "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
			},
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	_, stderr, exitCode, err := task.RunCommand(ctx, []string{"echo", "hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	// Start command assertions
	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "running",
			},
		},
	}, nil)

	err = task.Start(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, task.GetState().Status)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestTaskRunCommandWhileStopped(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	createResp := container.CreateResponse{ID: testContainerID}
	mockDocker.On("ContainerCreate", ctx, &container.Config{
		Image:      "nginx:latest",
		Entrypoint: []string{"sh", "-c"},
		Cmd:        []string{"sleep 36000"},
		Tty:        false,
		Hostname:   "test-task",
		Labels: map[string]string{
			providerLabelName: "test-provider",
		},
		Env: []string{},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/docker_volumes",
				Target: "",
			},
		},
		NetworkMode: container.NetworkMode("host"),
	}, (*network.NetworkingConfig)(nil), (*specs.Platform)(nil), mock.Anything).Return(createResp, nil)
	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)
	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
			},
		},
	}, nil).Twice()

	mockDO.On("GetDroplet", ctx, 1).Return(testDroplet, nil)

	execCreateResp := types.IDResponse{ID: "test-exec-id"}
	mockDocker.On("ContainerExecCreate", ctx, testContainerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"echo", "hello"},
	}).Return(execCreateResp, nil)

	conn := &mockConn{Buffer: bytes.NewBuffer([]byte{})}
	mockDocker.On("ContainerExecAttach", ctx, "test-exec-id", container.ExecAttachOptions{}).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}, nil)
	mockDocker.On("ContainerExecInspect", ctx, "test-exec-id").Return(container.ExecInspect{
		ExitCode: 0,
		Running:  false,
	}, nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: false,
			},
		},
	}, nil).Once()

	mockDocker.On("ContainerRemove", ctx, testContainerID, container.RemoveOptions{Force: true}).Return(nil)

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(1),
			Name:         "test-task",
			ProviderName: "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
			},
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	_, stderr, exitCode, err := task.RunCommand(ctx, []string{"echo", "hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	// Start command assertions
	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "running",
			},
		},
	}, nil)

	err = task.Start(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, task.GetState().Status)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestTaskGetIP(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscaleClient.On("Status", ctx).Return(generateTailscaleStatus(t, "test-task", "1.2.3.4"), nil)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	expectedIP := "1.2.3.4"
	droplet := &godo.Droplet{
		ID:     123,
		Status: "active",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{
					Type:      "public",
					IPAddress: "10.0.0.1",
				},
			},
		},
	}

	task := &Task{
		state: &TaskState{
			TailscaleHostname: "test-task",
			ID:                strconv.Itoa(droplet.ID),
			Name:              "test-task",
			ProviderName:      "test-provider",
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	ip, err := task.GetIP(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedIP, ip)

	externalAddr, err := task.GetExternalAddress(ctx, "80")
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf("%s:80", expectedIP), externalAddr)

	mockDO.AssertExpectations(t)
}

func TestTaskDestroy(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	mockDO.On("GetDroplet", ctx, testDroplet.ID).Return(testDroplet, nil)
	mockDO.On("DeleteDropletByID", ctx, testDroplet.ID).Return(nil)
	mockDocker.On("Close").Return(nil)

	provider := &Provider{
		state: &ProviderState{
			TaskStates: make(map[string]*TaskState),
		},
	}
	providerState := provider.GetState()

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(testDroplet.ID),
			Name:         "test-task",
			ProviderName: "test-provider",
		},
		logger:       logger,
		dockerClient: mockDocker,
		doClient:     mockDO,
		removeTask: func(ctx context.Context, taskID string) error {
			delete(provider.state.TaskStates, taskID)
			return nil
		},
		tailscaleSettings: mockTailscale,
	}

	providerState.TaskStates[task.GetState().ID] = task.state

	err := task.Destroy(ctx)
	require.NoError(t, err)
	require.Empty(t, providerState.TaskStates)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestRunCommandWhileStoppedContainerCleanup(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	mockDO.On("GetDroplet", ctx, 1).Return(testDroplet, nil)

	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{
		{ID: testContainerID},
	}, nil).Once()
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "exited",
			},
		},
	}, nil).Once()

	createResp := container.CreateResponse{ID: testContainerID}
	mockDocker.On("ContainerCreate", ctx, &container.Config{
		Image:      "nginx:latest",
		Entrypoint: []string{"sh", "-c"},
		Cmd:        []string{"sleep 36000"},
		Tty:        false,
		Hostname:   "test-task",
		Labels: map[string]string{
			providerLabelName: "test-provider",
		},
		Env: []string{},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/docker_volumes",
				Target: "",
			},
		},
		NetworkMode: container.NetworkMode("host"),
	}, (*network.NetworkingConfig)(nil), (*specs.Platform)(nil), mock.Anything).Return(createResp, nil)
	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)

	// first ContainerInspect for startup check
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
			},
		},
	}, nil).Once()

	execCreateResp := types.IDResponse{ID: "test-exec-id"}
	mockDocker.On("ContainerExecCreate", ctx, testContainerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"echo", "hello"},
	}).Return(execCreateResp, nil)

	conn := &mockConn{Buffer: bytes.NewBuffer([]byte{})}
	mockDocker.On("ContainerExecAttach", ctx, "test-exec-id", container.ExecAttachOptions{}).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}, nil)
	mockDocker.On("ContainerExecInspect", ctx, "test-exec-id").Return(container.ExecInspect{
		ExitCode: 0,
		Running:  false,
	}, nil)

	// second ContainerInspect for cleanup check - container exists
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{}, nil).Once()

	// container should be removed since it exists
	mockDocker.On("ContainerRemove", ctx, testContainerID, container.RemoveOptions{Force: true}).Return(nil)

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(1),
			Name:         "test-task",
			ProviderName: "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
			},
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	_, stderr, exitCode, err := task.RunCommand(ctx, []string{"echo", "hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

// this tests the case where the docker container is auto removed before cleanup doesnt return an error
func TestRunCommandWhileStoppedContainerAutoRemoved(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	mockDO.On("GetDroplet", ctx, 1).Return(testDroplet, nil)

	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{
		{ID: testContainerID},
	}, nil).Once()
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "exited",
			},
		},
	}, nil).Once()

	createResp := container.CreateResponse{ID: testContainerID}
	mockDocker.On("ContainerCreate", ctx, &container.Config{
		Image:      "nginx:latest",
		Entrypoint: []string{"sh", "-c"},
		Cmd:        []string{"sleep 36000"},
		Tty:        false,
		Hostname:   "test-task",
		Labels: map[string]string{
			providerLabelName: "test-provider",
		},
		Env: []string{},
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/docker_volumes",
				Target: "",
			},
		},
		NetworkMode: container.NetworkMode("host"),
	}, (*network.NetworkingConfig)(nil), (*specs.Platform)(nil), mock.Anything).Return(createResp, nil)
	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)

	// first ContainerInspect for startup check
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Running: true,
			},
		},
	}, nil).Once()

	execCreateResp := types.IDResponse{ID: "test-exec-id"}
	mockDocker.On("ContainerExecCreate", ctx, testContainerID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          []string{"echo", "hello"},
	}).Return(execCreateResp, nil)

	conn := &mockConn{Buffer: bytes.NewBuffer([]byte{})}
	mockDocker.On("ContainerExecAttach", ctx, "test-exec-id", container.ExecAttachOptions{}).Return(types.HijackedResponse{
		Conn:   conn,
		Reader: bufio.NewReader(conn),
	}, nil)
	mockDocker.On("ContainerExecInspect", ctx, "test-exec-id").Return(container.ExecInspect{
		ExitCode: 0,
		Running:  false,
	}, nil)

	// second ContainerInspect for cleanup check - container not found, so ContainerRemove should not be called
	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{}, mockNotFoundError{fmt.Errorf("Error: No such container: test-container-id")}).Once()
	mockDocker.AssertNotCalled(t, "ContainerRemove", ctx, testContainerID, mock.Anything)

	task := &Task{
		state: &TaskState{
			ID:           strconv.Itoa(1),
			Name:         "test-task",
			ProviderName: "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
			},
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	_, stderr, exitCode, err := task.RunCommand(ctx, []string{"echo", "hello"})
	require.NoError(t, err)
	require.Equal(t, 0, exitCode)
	require.Empty(t, stderr)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestTaskExposingPort(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	mockDocker := clientmocks.NewMockDockerClient(t)
	mockDO := mocks.NewMockDoClient(t)
	mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
	mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

	mockTailscaleClient.On("Status", ctx).Return(generateTailscaleStatus(t, "test-task", "1.2.3.4"), nil)

	mockTailscale := TailscaleSettings{
		Server:      mockTailscaleServer,
		LocalClient: mockTailscaleClient,
		AuthKey:     "test-auth-key",
		Tags:        []string{"test-tag"},
	}

	droplet := &godo.Droplet{
		ID:     123,
		Status: "active",
		Networks: &godo.Networks{
			V4: []godo.NetworkV4{
				{
					Type:      "public",
					IPAddress: "10.0.0.1",
				},
			},
		},
	}

	mockDO.On("GetDroplet", ctx, droplet.ID).Return(droplet, nil)

	testContainer := types.Container{
		ID: testContainerID,
		Ports: []types.Port{
			{
				PrivatePort: 80,
				PublicPort:  80,
				Type:        "tcp",
			},
		},
	}
	mockDocker.On("ContainerList", ctx, container.ListOptions{
		Limit: 1,
	}).Return([]types.Container{testContainer}, nil)

	mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
		ContainerJSONBase: &types.ContainerJSONBase{
			State: &types.ContainerState{
				Status: "running",
			},
		},
	}, nil)

	mockDocker.On("ContainerStart", ctx, testContainerID, container.StartOptions{}).Return(nil)

	task := &Task{
		state: &TaskState{
			TailscaleHostname: "test-task",
			ID:                strconv.Itoa(droplet.ID),
			Name:              "test-task",
			ProviderName:      "test-provider",
			Definition: provider.TaskDefinition{
				Name: "test-task",
				Image: provider.ImageDefinition{
					Image: "nginx:latest",
					UID:   "1000",
					GID:   "1000",
				},
				Ports: []string{"80"},
			},
			Status: provider.TASK_STOPPED,
		},
		logger:            logger,
		dockerClient:      mockDocker,
		doClient:          mockDO,
		tailscaleSettings: mockTailscale,
	}

	err := task.Start(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, task.GetState().Status)

	externalAddr, err := task.GetExternalAddress(ctx, "80")
	require.NoError(t, err)
	require.Equal(t, "1.2.3.4:80", externalAddr)

	status, err := task.GetStatus(ctx)
	require.NoError(t, err)
	require.Equal(t, provider.TASK_RUNNING, status)

	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s", externalAddr), nil)
	require.NoError(t, err)
	require.NotEmpty(t, req)

	mockDocker.AssertExpectations(t)
	mockDO.AssertExpectations(t)
}

func TestGetStatus(t *testing.T) {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()
	testDropletActive := &godo.Droplet{
		ID:     123,
		Status: "active",
	}
	testDropletOff := &godo.Droplet{
		ID:     123,
		Status: "off",
	}

	tests := []struct {
		name           string
		dropletStatus  string
		containerState string
		setupMocks     func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings)
		expectedStatus provider.TaskStatus
		expectError    bool
	}{
		{
			name:           "droplet not active",
			dropletStatus:  "off",
			containerState: "",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletOff.ID).Return(testDropletOff, nil)
			},
			expectedStatus: provider.TASK_STOPPED,
			expectError:    false,
		},
		{
			name:           "container running",
			dropletStatus:  "active",
			containerState: "running",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "running",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_RUNNING,
			expectError:    false,
		},
		{
			name:           "container paused",
			dropletStatus:  "active",
			containerState: "paused",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "paused",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_PAUSED,
			expectError:    false,
		},
		{
			name:           "container stopped state",
			dropletStatus:  "active",
			containerState: "exited",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "exited",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_STOPPED,
			expectError:    false,
		},
		{
			name:           "container removing",
			dropletStatus:  "active",
			containerState: "removing",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "removing",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_STOPPED,
			expectError:    false,
		},
		{
			name:           "container dead",
			dropletStatus:  "active",
			containerState: "dead",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "dead",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_STOPPED,
			expectError:    false,
		},
		{
			name:           "container created",
			dropletStatus:  "active",
			containerState: "created",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "created",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_STOPPED,
			expectError:    false,
		},
		{
			name:           "unknown container status",
			dropletStatus:  "active",
			containerState: "unknown_status",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{
						State: &types.ContainerState{
							Status: "unknown_status",
						},
					},
				}, nil)
			},
			expectedStatus: provider.TASK_STATUS_UNDEFINED,
			expectError:    false,
		},
		{
			name:           "no containers found",
			dropletStatus:  "active",
			containerState: "",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{}, nil)
			},
			expectedStatus: provider.TASK_STATUS_UNDEFINED,
			expectError:    true,
		},
		{
			name:           "container inspect error",
			dropletStatus:  "active",
			containerState: "",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return([]types.Container{testContainer}, nil)
				mockDocker.On("ContainerInspect", ctx, testContainerID).Return(types.ContainerJSON{}, fmt.Errorf("inspect error"))
			},
			expectedStatus: provider.TASK_STATUS_UNDEFINED,
			expectError:    true,
		},
		{
			name:           "GetDroplet error",
			dropletStatus:  "",
			containerState: "",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, 123).Return(nil, fmt.Errorf("failed to get droplet"))
			},
			expectedStatus: provider.TASK_STATUS_UNDEFINED,
			expectError:    true,
		},
		{
			name:           "containerList error",
			dropletStatus:  "active",
			containerState: "",
			setupMocks: func(mockDocker *clientmocks.MockDockerClient, mockDO *mocks.MockDoClient, tailscale *TailscaleSettings) {
				mockDO.On("GetDroplet", ctx, testDropletActive.ID).Return(testDropletActive, nil)
				mockDocker.On("ContainerList", ctx, container.ListOptions{
					Limit: 1,
				}).Return(nil, fmt.Errorf("failed to list containers"))
			},
			expectedStatus: provider.TASK_STATUS_UNDEFINED,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockDocker := clientmocks.NewMockDockerClient(t)
			mockDO := mocks.NewMockDoClient(t)
			mockTailscaleServer := clientmocks.NewMockTailscaleServer(t)
			mockTailscaleClient := clientmocks.NewMockTailscaleLocalClient(t)

			mockTailscale := TailscaleSettings{
				Server:      mockTailscaleServer,
				LocalClient: mockTailscaleClient,
				AuthKey:     "test-auth-key",
				Tags:        []string{"test-tag"},
			}

			tt.setupMocks(mockDocker, mockDO, &mockTailscale)

			task := &Task{
				state: &TaskState{
					ID:           strconv.Itoa(123),
					Name:         "test-task",
					ProviderName: "test-provider",
					Definition: provider.TaskDefinition{
						Name: "test-task",
					},
				},
				logger:       logger,
				dockerClient: mockDocker,
				doClient:     mockDO,
			}

			status, err := task.GetStatus(ctx)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.expectedStatus, status)

			mockDocker.AssertExpectations(t)
			mockDO.AssertExpectations(t)
		})
	}
}
