package digitalocean

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"path"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerclient "github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/pkg/sftp"
	"github.com/spf13/afero"
	"github.com/spf13/afero/sftpfs"
	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/clients"
	"github.com/skip-mev/ironbird/petri/core/util"
)

type TaskState struct {
	ID                string                  `json:"id"`
	Name              string                  `json:"name"`
	TailscaleHostname string                  `json:"tailscale_hostname"`
	Definition        provider.TaskDefinition `json:"definition"`
	Status            provider.TaskStatus     `json:"status"`
	ProviderName      string                  `json:"provider_name"`
}

type Task struct {
	state   *TaskState
	stateMu sync.Mutex

	removeTask        provider.RemoveTaskFunc
	logger            *zap.Logger
	sshClient         *ssh.Client
	doClient          DoClient
	dockerClient      clients.DockerClient
	tailscaleSettings TailscaleSettings
}

var _ provider.TaskI = (*Task)(nil)

func (t *Task) Start(ctx context.Context) error {
	containers, err := t.dockerClient.ContainerList(ctx, container.ListOptions{
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf("failed to retrieve containers: %w", err)
	}

	if len(containers) != 1 {
		return fmt.Errorf("could not find container for %s", t.GetState().Name)
	}

	containerID := containers[0].ID

	err = t.dockerClient.ContainerStart(ctx, containerID, container.StartOptions{})
	if err != nil {
		return err
	}

	err = util.WaitForCondition(ctx, time.Second*300, time.Millisecond*100, func() (bool, error) {
		status, err := t.GetStatus(ctx)
		if err != nil {
			return false, err
		}

		if status != provider.TASK_RUNNING {
			return false, nil
		}

		t.stateMu.Lock()
		defer t.stateMu.Unlock()

		t.state.Status = provider.TASK_RUNNING
		return true, nil
	})

	t.logger.Info("final task status after start", zap.Any("status", t.GetState().Status))
	return err
}

func (t *Task) Stop(ctx context.Context) error {
	containers, err := t.dockerClient.ContainerList(ctx, container.ListOptions{
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf("failed to retrieve containers: %w", err)
	}

	if len(containers) != 1 {
		return fmt.Errorf("could not find container for %s", t.GetState().Name)
	}

	t.stateMu.Lock()
	defer t.stateMu.Unlock()

	t.state.Status = provider.TASK_STOPPED
	return t.dockerClient.ContainerStop(ctx, containers[0].ID, container.StopOptions{})
}

func (t *Task) Modify(ctx context.Context, definition provider.TaskDefinition) error {
	panic("implement me")
}

func (t *Task) Destroy(ctx context.Context) error {
	logger := t.logger.With(zap.String("task", t.GetState().Name))
	logger.Info("deleting task")
	defer t.dockerClient.Close()

	err := t.deleteDroplet(ctx)
	if err != nil {
		return err
	}

	if err := t.removeTask(ctx, t.GetState().ID); err != nil {
		return err
	}
	return nil
}

func (t *Task) GetState() TaskState {
	t.stateMu.Lock()
	defer t.stateMu.Unlock()
	return *t.state
}

func (t *Task) GetDefinition() provider.TaskDefinition {
	return t.GetState().Definition
}

func (t *Task) GetStatus(ctx context.Context) (provider.TaskStatus, error) {
	droplet, err := t.GetDroplet(ctx)
	if err != nil {
		return provider.TASK_STATUS_UNDEFINED, err
	}

	if droplet.Status != "active" {
		return provider.TASK_STOPPED, nil
	}

	containers, err := t.dockerClient.ContainerList(ctx, container.ListOptions{
		Limit: 1,
	})

	if err != nil {
		return provider.TASK_STATUS_UNDEFINED, fmt.Errorf("failed to retrieve containers: %w", err)
	}

	if len(containers) != 1 {
		return provider.TASK_STATUS_UNDEFINED, fmt.Errorf("could not find container for %s", t.GetState().Name)
	}

	c, err := t.dockerClient.ContainerInspect(ctx, containers[0].ID)
	if err != nil {
		return provider.TASK_STATUS_UNDEFINED, err
	}

	switch state := c.State.Status; state {
	case "created":
		return provider.TASK_STOPPED, nil
	case "running":
		return provider.TASK_RUNNING, nil
	case "paused":
		return provider.TASK_PAUSED, nil
	case "removing":
		return provider.TASK_STOPPED, nil
	case "restarting":
		return provider.TASK_RESTARTING, nil
	case "exited":
		return provider.TASK_STOPPED, nil
	case "dead":
		return provider.TASK_STOPPED, nil
	}

	return provider.TASK_STATUS_UNDEFINED, nil
}

func (t *Task) WriteFile(ctx context.Context, relPath string, content []byte) error {
	absPath := path.Join("/docker_volumes", relPath)

	sshClient, err := t.getDropletSSHClient(ctx)
	if err != nil {
		return err
	}

	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return err
	}

	defer sftpClient.Close()

	err = sftpClient.MkdirAll(path.Dir(absPath))
	if err != nil {
		return err
	}

	file, err := sftpClient.Create(absPath)
	if err != nil {
		return err
	}

	_, err = file.Write(content)
	if err != nil {
		return err
	}

	return nil
}

func (t *Task) ReadFile(ctx context.Context, relPath string) ([]byte, error) {
	absPath := path.Join("/docker_volumes", relPath)

	sshClient, err := t.getDropletSSHClient(ctx)
	if err != nil {
		return nil, err
	}

	defer sshClient.Close()

	sftpClient, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, err
	}

	defer sftpClient.Close()

	fs := sftpfs.New(sftpClient)

	content, err := afero.ReadFile(fs, absPath)
	if err != nil {
		return nil, err
	}

	return content, nil
}

func (t *Task) DownloadDir(ctx context.Context, s string, s2 string) error {
	panic("implement me")
}

// GetIP returns *Tailscale* IP.
// todo: STACK-1616: refactor to GetIP(ctx, type=[private, tailscale, public, ...])
func (t *Task) GetIP(ctx context.Context) (string, error) {
	return t.getTailscaleIp(ctx)
}

// GetPrivateIP returns node's private IP address
func (t *Task) GetPrivateIP(ctx context.Context) (string, error) {
	droplet, err := t.GetDroplet(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get droplet: %w", err)
	}

	return droplet.PrivateIPv4()
}

func (t *Task) GetExternalAddress(ctx context.Context, port string) (string, error) {
	ip, err := t.GetIP(ctx)
	if err != nil {
		return "", err
	}

	return net.JoinHostPort(ip, port), nil
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

func waitForExec(ctx context.Context, dockerClient clients.DockerClient, execID string) (int, error) {
	lastExitCode := 0
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ctx.Done():
			return lastExitCode, ctx.Err()
		case <-ticker.C:
			execInspect, err := dockerClient.ContainerExecInspect(ctx, execID)
			if err != nil {
				return lastExitCode, err
			}

			if execInspect.Running {
				continue
			}

			lastExitCode = execInspect.ExitCode
			break loop
		}
	}

	return lastExitCode, nil
}

func (t *Task) runCommand(ctx context.Context, command []string) (string, string, int, error) {
	containers, err := t.dockerClient.ContainerList(ctx, container.ListOptions{
		Limit: 1,
	})

	if err != nil {
		return "", "", 0, fmt.Errorf("failed to retrieve containers: %w", err)
	}

	if len(containers) != 1 {
		return "", "", 0, fmt.Errorf("could not find container for %s", t.GetState().Name)
	}

	id := containers[0].ID

	t.logger.Debug("running command", zap.String("id", id), zap.Strings("command", command))

	exec, err := t.dockerClient.ContainerExecCreate(ctx, id, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          command,
	})
	if err != nil {
		return "", "", 0, err
	}

	resp, err := t.dockerClient.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		return "", "", 0, err
	}

	defer resp.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", "", 0, err
	}

	exitCode, err := waitForExec(ctx, t.dockerClient, exec.ID)
	if err != nil {
		return stdout.String(), stderr.String(), exitCode, err
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

func (t *Task) runCommandWhileStopped(ctx context.Context, cmd []string) (string, string, int, error) {
	state := t.GetState()
	if err := state.Definition.ValidateBasic(); err != nil {
		return "", "", 0, fmt.Errorf("failed to validate task definition: %w", err)
	}

	containerName := fmt.Sprintf("%s-executor-%s-%d", state.Definition.Name, util.RandomString(5), time.Now().Unix())
	createdContainer, err := t.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:      state.Definition.Image.Image,
		Entrypoint: []string{"sh", "-c"},
		Cmd:        []string{"sleep 36000"},
		Tty:        false,
		Hostname:   state.Definition.Name,
		Labels: map[string]string{
			providerLabelName: state.ProviderName,
		},
		Env: convertEnvMapToList(state.Definition.Environment),
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/docker_volumes",
				Target: state.Definition.DataDir,
			},
		},
		NetworkMode: container.NetworkMode("host"),
	}, nil, nil, containerName)
	if err != nil {
		t.logger.Error("failed to create container", zap.Error(err), zap.String("taskName", state.Name))
		return "", "", 0, err
	}

	t.logger.Debug("container created successfully", zap.String("id", createdContainer.ID), zap.String("taskName", state.Name))

	defer func() {
		if _, err := t.dockerClient.ContainerInspect(ctx, createdContainer.ID); err != nil && dockerclient.IsErrNotFound(err) {
			// container was auto-removed, no need to remove it again
			return
		}

		if err := t.dockerClient.ContainerRemove(ctx, createdContainer.ID, container.RemoveOptions{Force: true}); err != nil {
			t.logger.Error("failed to remove container", zap.Error(err), zap.String("taskName", state.Name), zap.String("id", createdContainer.ID))
		}
	}()

	if err := startContainerWithBlock(ctx, t.dockerClient, createdContainer.ID); err != nil {
		t.logger.Error("failed to start container", zap.Error(err), zap.String("taskName", state.Name))
		return "", "", 0, err
	}

	t.logger.Debug("container started successfully", zap.String("id", createdContainer.ID), zap.String("taskName", state.Name))

	// wait for container start
	exec, err := t.dockerClient.ContainerExecCreate(ctx, createdContainer.ID, container.ExecOptions{
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		t.logger.Error("failed to create exec", zap.Error(err), zap.String("taskName", state.Name))
		return "", "", 0, err
	}

	resp, err := t.dockerClient.ContainerExecAttach(ctx, exec.ID, container.ExecAttachOptions{})
	if err != nil {
		t.logger.Error("failed to attach to exec", zap.Error(err), zap.String("taskName", state.Name))
		return "", "", 0, err
	}

	defer resp.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, resp.Reader)
	if err != nil {
		return "", "", 0, err
	}

	exitCode, err := waitForExec(ctx, t.dockerClient, exec.ID)
	if err != nil {
		return stdout.String(), stderr.String(), exitCode, err
	}

	return stdout.String(), stderr.String(), exitCode, nil
}

func startContainerWithBlock(ctx context.Context, dockerClient clients.DockerClient, containerID string) error {
	// start container
	if err := dockerClient.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return err
	}

	// cancel container after a minute
	waitCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-waitCtx.Done():
			return fmt.Errorf("error waiting for container to start: %v", waitCtx.Err())
		case <-ticker.C:
			container, err := dockerClient.ContainerInspect(ctx, containerID)
			if err != nil {
				return err
			}

			// if the container is running, we're done
			if container.State.Running {
				return nil
			}

			if container.State.Status == "exited" && container.State.ExitCode != 0 {
				return fmt.Errorf("container exited with status %d", container.State.ExitCode)
			}
		}
	}
}

func (t *Task) DialContext() func(ctx context.Context, network, address string) (net.Conn, error) {
	return t.tailscaleSettings.Server.Dial
}
