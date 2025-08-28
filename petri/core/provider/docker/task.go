package docker

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/docker/go-connections/nat"
	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/clients"
	"go.uber.org/zap"
)

type TaskState struct {
	Id               string                  `json:"id"`
	Name             string                  `json:"name"`
	Volume           *VolumeState            `json:"volumes"`
	Definition       provider.TaskDefinition `json:"definition"`
	Status           provider.TaskStatus     `json:"status"`
	IpAddress        string                  `json:"ip_address"`
	BuilderImageName string                  `json:"builder_image_name"`
	NetworkName      string                  `json:"network_name"`
	PortBindings     nat.PortMap             `json:"port_bindings"`
}

type VolumeState struct {
	Name string `json:"name"`
	Size string `json:"size"`
}

type Task struct {
	state        *TaskState
	stateMu      sync.Mutex
	logger       *zap.Logger
	dockerClient clients.DockerClient
	removeTask   provider.RemoveTaskFunc
}

var _ provider.TaskI = (*Task)(nil)

func (t *Task) Start(ctx context.Context) error {
	state := t.GetState()
	t.logger.Info("starting task", zap.String("id", state.Id))

	err := t.dockerClient.ContainerStart(ctx, state.Id, container.StartOptions{})
	if err != nil {
		return err
	}

	if err := t.WaitForStatus(ctx, 1*time.Second, provider.TASK_RUNNING); err != nil {
		return err
	}

	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	t.state.Status = provider.TASK_RUNNING

	return nil
}

func (t *Task) Stop(ctx context.Context) error {
	state := t.GetState()
	t.logger.Info("stopping task", zap.String("id", state.Id))

	err := t.dockerClient.ContainerStop(ctx, state.Id, container.StopOptions{})
	if err != nil {
		return err
	}

	if err := t.WaitForStatus(ctx, 1*time.Second, provider.TASK_STOPPED); err != nil {
		return err
	}

	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	t.state.Status = provider.TASK_STOPPED

	return nil
}

func (t *Task) Destroy(ctx context.Context) error {
	state := t.GetState()
	t.logger.Info("destroying task", zap.String("id", state.Id))

	err := t.dockerClient.ContainerRemove(ctx, state.Id, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})

	if err != nil {
		return err
	}

	if err := t.removeTask(ctx, state.Id); err != nil {
		return err
	}

	return nil
}

func (t *Task) GetExternalAddress(ctx context.Context, port string) (string, error) {
	state := t.GetState()

	portKey := nat.Port(fmt.Sprintf("%s/tcp", port))
	portBindings, ok := state.PortBindings[portKey]
	if !ok || len(portBindings) == 0 {
		return "", fmt.Errorf("port %s not found", port)
	}

	return fmt.Sprintf("0.0.0.0:%s", portBindings[0].HostPort), nil
}

func (t *Task) GetIP(ctx context.Context) (string, error) {
	state := t.GetState()
	t.logger.Debug("getting IP", zap.String("id", state.Id))

	dockerContainer, err := t.dockerClient.ContainerInspect(ctx, state.Id)
	if err != nil {
		return "", err
	}

	ip := dockerContainer.NetworkSettings.Networks[state.NetworkName].IPAMConfig.IPv4Address
	return ip, nil
}

func (t *Task) WaitForStatus(ctx context.Context, interval time.Duration, desiredStatus provider.TaskStatus) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			status, err := t.GetStatus(ctx)
			if err != nil {
				return err
			}

			if status == desiredStatus {
				return nil
			}
			time.Sleep(interval)
		}
	}
}

func (t *Task) GetStatus(ctx context.Context) (provider.TaskStatus, error) {
	containerJSON, err := t.dockerClient.ContainerInspect(ctx, t.GetState().Id)
	if err != nil {
		return provider.TASK_STATUS_UNDEFINED, err
	}

	switch state := containerJSON.State.Status; state {
	case "created":
		return provider.TASK_STOPPED, nil
	case "running":
		return provider.TASK_RUNNING, nil
	case "paused":
		return provider.TASK_PAUSED, nil
	case "restarting":
		return provider.TASK_RESTARTING, nil
	case "removing":
		return provider.TASK_STOPPED, nil
	case "exited":
		return provider.TASK_STOPPED, nil
	case "dead":
		return provider.TASK_STOPPED, nil
	}

	return provider.TASK_STATUS_UNDEFINED, nil
}

func (t *Task) Modify(ctx context.Context, td provider.TaskDefinition) error {
	panic("unimplemented")
}

func (t *Task) RunCommand(ctx context.Context, cmd []string) (string, string, int, error) {
	status, err := t.GetStatus(ctx)
	if err != nil {
		return "", "", 0, err
	}

	if status != provider.TASK_RUNNING {
		return t.runCommandWhileStopped(ctx, cmd)
	}

	return t.runCommand(ctx, cmd)
}

func (t *Task) runCommand(ctx context.Context, cmd []string) (string, string, int, error) {
	state := t.GetState()
	t.logger.Debug("running command", zap.String("id", state.Id), zap.Strings("command", cmd))

	exec, err := t.dockerClient.ContainerExecCreate(ctx, state.Id, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		if buf, err := t.dockerClient.ContainerLogs(ctx, state.Id, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
		}); err == nil {
			defer buf.Close()
			var out bytes.Buffer
			_, _ = out.ReadFrom(buf)
			fmt.Println(out.String())
		}

		return "", "", 0, err
	}

	resp, err := t.dockerClient.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", "", 0, err
	}

	defer resp.Close()

	lastExitCode := 0

	//TODO(Zygimantass): talk with Eric about best practices here
	ticker := time.NewTicker(1 * time.Second)

	//TODO(Zygimantass): i hate this
loop:
	for {
		select {
		case <-ctx.Done():
			return "", "", lastExitCode, ctx.Err()
		case <-ticker.C:
			execInspect, err := t.dockerClient.ContainerExecInspect(ctx, exec.ID)
			if err != nil {
				return "", "", lastExitCode, err
			}

			if execInspect.Running {
				continue
			}

			lastExitCode = execInspect.ExitCode
			break loop
		}
	}

	if err != nil {
		t.logger.Error("failed to wait for exec", zap.Error(err), zap.String("id", t.GetState().Id))
		return "", "", lastExitCode, err
	}

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", "", lastExitCode, err
	}

	return stdout.String(), stderr.String(), lastExitCode, nil
}

func (t *Task) runCommandWhileStopped(ctx context.Context, cmd []string) (string, string, int, error) {
	state := t.GetState()
	definition := t.GetState().Definition
	if err := definition.ValidateBasic(); err != nil {
		return "", "", 0, fmt.Errorf("failed to validate task definition: %w", err)
	}

	t.logger.Debug("running command while stopped", zap.String("id", state.Id), zap.Strings("command", cmd))

	status, err := t.GetStatus(ctx)
	if err != nil {
		return "", "", 0, err
	}

	if status == provider.TASK_RUNNING {
		return t.RunCommand(ctx, cmd)
	}

	definition.Entrypoint = []string{"/bin/sh", "-c"}
	definition.Command = []string{"sleep 36000"}
	definition.Name = fmt.Sprintf("%s-exec-%d", state.Name, time.Now().Unix()%3000)
	definition.Ports = []string{}

	containerConfig := &container.Config{
		Image:      definition.Image.Image,
		Entrypoint: definition.Entrypoint,
		Cmd:        definition.Command,
		Tty:        false,
		Hostname:   definition.Name,
		Env:        convertEnvMapToList(definition.Environment),
	}

	var mounts []mount.Mount
	if state.Volume != nil {
		mounts = []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: state.Volume.Name,
				Target: definition.DataDir,
			},
		}
	}

	hostConfig := &container.HostConfig{
		NetworkMode: container.NetworkMode(state.NetworkName),
		Mounts:      mounts,
	}

	resp, err := t.dockerClient.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, definition.Name)
	if err != nil {
		return "", "", 0, err
	}

	tempTask := &Task{
		state: &TaskState{
			Id:          resp.ID,
			Name:        definition.Name,
			Definition:  definition,
			Status:      provider.TASK_STOPPED,
			NetworkName: state.NetworkName,
		},
		logger:       t.logger.With(zap.String("temp_task", definition.Name)),
		dockerClient: t.dockerClient,
		removeTask:   t.removeTask,
	}

	err = tempTask.Start(ctx)
	if err != nil {
		return "", "", 0, err
	}

	defer func() {
		if err := tempTask.Destroy(ctx); err != nil {
			t.logger.Error("failed to destroy temporary task", zap.Error(err))
		}
	}()

	return tempTask.RunCommand(ctx, cmd)
}

func (t *Task) GetState() TaskState {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return *t.state
}

func (t *Task) ensureTask(ctx context.Context) error {
	state := t.GetState()

	dockerContainer, err := t.dockerClient.ContainerInspect(ctx, state.Id)
	if err != nil {
		return fmt.Errorf("failed to inspect container: %w", err)
	}

	actualStatus, err := t.GetStatus(ctx)

	if err != nil {
		return fmt.Errorf("failed to get task status: %w", err)
	}

	if actualStatus != state.Status {
		return fmt.Errorf("task status mismatch, expected: %d, got: %d", state.Status, actualStatus)
	}

	// ref: https://github.com/moby/moby/issues/6705
	if dockerContainer.Name != fmt.Sprintf("/%s", state.Name) {
		return fmt.Errorf("task name mismatch, expected: %s, got: %s", state.Name, dockerContainer.Name)
	}

	if err := t.ensureVolume(ctx); err != nil {
		return err
	}

	return nil
}

func (t *Task) ensureVolume(ctx context.Context) error {
	state := t.GetState()
	if state.Volume == nil {
		return nil
	}

	volume, err := t.dockerClient.VolumeInspect(ctx, state.Volume.Name)
	if err != nil {
		return fmt.Errorf("failed to inspect volume: %w", err)
	}

	if volume.Name != state.Volume.Name {
		return fmt.Errorf("volume name mismatch, expected: %s, got: %s", state.Volume.Name, volume.Name)
	}

	return nil
}

func (t *Task) GetDefinition() provider.TaskDefinition {
	return t.GetState().Definition
}

func (t *Task) DialContext() func(context.Context, string, string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext
}
