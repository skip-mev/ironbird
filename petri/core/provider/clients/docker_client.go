package clients

import (
	"context"
	"fmt"
	"io"
	"net"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"go.uber.org/zap"
)

// DockerClient is a unified interface for interacting with Docker
// It combines functionality needed by both the Docker and DigitalOcean providers
type DockerClient interface {
	// Container Operations
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, container string, options container.StartOptions) error
	ContainerStop(ctx context.Context, container string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, container string, options container.RemoveOptions) error
	ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error)
	ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error)
	ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error)

	// Container Exec Operations
	ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error)
	ContainerExecAttach(ctx context.Context, execID string, config container.ExecStartOptions) (types.HijackedResponse, error)
	ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error)

	// Container File Operations
	CopyToContainer(ctx context.Context, container, path string, content io.Reader, options container.CopyToContainerOptions) error
	CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error)
	ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error)

	// Image Operations
	ImageInspectWithRaw(ctx context.Context, imageID string) (types.ImageInspect, []byte, error)
	ImagePull(ctx context.Context, logger *zap.Logger, refStr string, options image.PullOptions) error

	// Volume Operations
	VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error)
	VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error)
	VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error)
	VolumeRemove(ctx context.Context, volumeID string, force bool) error

	// Network Operations
	NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error)
	NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error)
	NetworkRemove(ctx context.Context, networkID string) error

	// System Operations
	Ping(ctx context.Context) (types.Ping, error)
	Close() error
}

// defaultDockerClient is the default implementation of DockerClient interface
type defaultDockerClient struct {
	client *dockerclient.Client
}

func NewDockerClient(host string, dialFunc func(ctx context.Context, network, address string) (net.Conn, error)) (DockerClient, error) {
	var opts []dockerclient.Opt

	if host != "" {
		host = fmt.Sprintf("tcp://%s:2375", host)
		opts = append(opts, dockerclient.WithHost(host))
	}

	if dialFunc != nil {
		opts = append(opts, dockerclient.WithDialContext(dialFunc))
	}

	client, err := dockerclient.NewClientWithOpts(opts...)
	if err != nil {
		return nil, err
	}
	return &defaultDockerClient{client: client}, nil
}

func (d *defaultDockerClient) Ping(ctx context.Context) (types.Ping, error) {
	return d.client.Ping(ctx)
}

func (d *defaultDockerClient) ImageInspectWithRaw(ctx context.Context, image string) (types.ImageInspect, []byte, error) {
	return d.client.ImageInspectWithRaw(ctx, image)
}

func (d *defaultDockerClient) ImagePull(ctx context.Context, logger *zap.Logger, ref string, options image.PullOptions) error {
	_, _, err := d.client.ImageInspectWithRaw(ctx, ref)
	if err != nil {
		logger.Info("pulling image", zap.String("image", ref))
		resp, err := d.client.ImagePull(ctx, ref, options)
		if err != nil {
			return fmt.Errorf("failed to pull docker image: %w", err)
		}

		defer resp.Close()
		// throw away the image pull stdout response
		_, err = io.Copy(io.Discard, resp)
		if err != nil {
			return fmt.Errorf("failed to pull docker image: %w", err)
		}
		return nil
	}
	return nil
}

func (d *defaultDockerClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) (container.CreateResponse, error) {
	return d.client.ContainerCreate(ctx, config, hostConfig, networkingConfig, platform, containerName)
}

func (d *defaultDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return d.client.ContainerList(ctx, options)
}

func (d *defaultDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return d.client.ContainerStart(ctx, containerID, options)
}

func (d *defaultDockerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	return d.client.ContainerStop(ctx, containerID, options)
}

func (d *defaultDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return d.client.ContainerInspect(ctx, containerID)
}

func (d *defaultDockerClient) ContainerExecCreate(ctx context.Context, container string, config container.ExecOptions) (types.IDResponse, error) {
	return d.client.ContainerExecCreate(ctx, container, config)
}

func (d *defaultDockerClient) ContainerExecAttach(ctx context.Context, execID string, config container.ExecStartOptions) (types.HijackedResponse, error) {
	return d.client.ContainerExecAttach(ctx, execID, config)
}

func (d *defaultDockerClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return d.client.ContainerExecInspect(ctx, execID)
}

func (d *defaultDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	return d.client.ContainerRemove(ctx, containerID, options)
}

func (d *defaultDockerClient) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return d.client.ContainerWait(ctx, containerID, condition)
}

func (d *defaultDockerClient) ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error) {
	return d.client.ContainerLogs(ctx, container, options)
}

func (d *defaultDockerClient) CopyToContainer(ctx context.Context, container, path string, content io.Reader, options container.CopyToContainerOptions) error {
	return d.client.CopyToContainer(ctx, container, path, content, options)
}

func (d *defaultDockerClient) CopyFromContainer(ctx context.Context, container, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return d.client.CopyFromContainer(ctx, container, srcPath)
}

func (d *defaultDockerClient) VolumeCreate(ctx context.Context, options volume.CreateOptions) (volume.Volume, error) {
	return d.client.VolumeCreate(ctx, options)
}

func (d *defaultDockerClient) VolumeInspect(ctx context.Context, volumeID string) (volume.Volume, error) {
	return d.client.VolumeInspect(ctx, volumeID)
}

func (d *defaultDockerClient) VolumeList(ctx context.Context, options volume.ListOptions) (volume.ListResponse, error) {
	return d.client.VolumeList(ctx, options)
}

func (d *defaultDockerClient) VolumeRemove(ctx context.Context, volumeID string, force bool) error {
	return d.client.VolumeRemove(ctx, volumeID, force)
}

func (d *defaultDockerClient) NetworkCreate(ctx context.Context, name string, options network.CreateOptions) (network.CreateResponse, error) {
	return d.client.NetworkCreate(ctx, name, options)
}

func (d *defaultDockerClient) NetworkInspect(ctx context.Context, networkID string, options network.InspectOptions) (network.Inspect, error) {
	return d.client.NetworkInspect(ctx, networkID, options)
}

func (d *defaultDockerClient) NetworkRemove(ctx context.Context, networkID string) error {
	return d.client.NetworkRemove(ctx, networkID)
}

func (d *defaultDockerClient) Close() error {
	return d.client.Close()
}
