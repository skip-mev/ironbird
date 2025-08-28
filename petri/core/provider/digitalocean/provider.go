package digitalocean

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/skip-mev/ironbird/petri/core/types"

	"github.com/digitalocean/godo"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"

	"go.uber.org/zap"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/clients"
	"github.com/skip-mev/ironbird/petri/core/util"
)

var _ provider.ProviderI = (*Provider)(nil)

const (
	providerLabelName = "petri-provider"
	portsLabelName    = "petri-ports"
	nodeNameLabelName = "petri-node-name"
)

type ProviderState struct {
	TaskStates map[string]*TaskState `json:"task_states"` // map of task ids to the corresponding task state
	Name       string                `json:"name"`
	PetriTag   string                `json:"petri_tag"`
	FirewallID string                `json:"firewall_id"`
	DomainIDs  []int                 `json:"domain_ids"`
}

type Provider struct {
	state   *ProviderState
	stateMu sync.Mutex

	logger                *zap.Logger
	doClient              DoClient
	tailscaleSettings     TailscaleSettings
	telemetrySettings     *TelemetrySettings
	domain                string
	dockerClientOverrides map[string]clients.DockerClient // map of droplet name to docker clients
}

func NewProvider(ctx context.Context, providerName, token string, tailscaleSettings TailscaleSettings, opts ...func(*Provider)) (*Provider, error) {
	if token == "" {
		return nil, errors.New("a non-empty token must be passed when creating a DigitalOcean provider")
	}

	doClient := NewGodoClient(token)
	return NewProviderWithClient(ctx, providerName, doClient, tailscaleSettings, opts...)
}

// NewProviderWithClient creates a DigitalOcean provider given an existing DigitalOcean client
// with additional options to configure behaviour.
func NewProviderWithClient(ctx context.Context, providerName string, doClient DoClient, tailscaleSettings TailscaleSettings, opts ...func(*Provider)) (*Provider, error) {
	if doClient == nil {
		return nil, errors.New("a valid digital ocean client must be passed when creating a provider")
	}

	if err := tailscaleSettings.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("failed to validate tailscale settings: %w", err)
	}

	petriTag := fmt.Sprintf("petri-%s", providerName)
	digitalOceanProvider := &Provider{
		doClient:          doClient,
		tailscaleSettings: tailscaleSettings,
		state: &ProviderState{
			TaskStates: make(map[string]*TaskState),
			Name:       providerName,
			PetriTag:   petriTag,
		},
	}

	for _, opt := range opts {
		opt(digitalOceanProvider)
	}

	if digitalOceanProvider.logger == nil {
		digitalOceanProvider.logger = zap.NewNop()
	}

	_, err := digitalOceanProvider.createTag(ctx, petriTag)
	if err != nil {
		return nil, err
	}

	firewall, err := digitalOceanProvider.createFirewall(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create firewall: %w", err)
	}

	digitalOceanProvider.state.FirewallID = firewall.ID

	return digitalOceanProvider, nil
}

func (p *Provider) CreateTask(ctx context.Context, definition provider.TaskDefinition) (provider.TaskI, error) {
	if err := definition.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("failed to validate task definition: %w", err)
	}

	if definition.ProviderSpecificConfig == nil {
		return nil, fmt.Errorf("digitalocean specific config is nil for %s", definition.Name)
	}

	var doConfig DigitalOceanTaskConfig = definition.ProviderSpecificConfig

	if err := doConfig.ValidateBasic(); err != nil {
		return nil, fmt.Errorf("could not cast digitalocean specific config: %w", err)
	}

	p.logger.Info("creating droplet", zap.String("name", definition.Name))

	droplet, err := p.CreateDroplet(ctx, definition)
	if err != nil {
		return nil, err
	}

	p.logger.Info("droplet created", zap.String("name", droplet.Name))

	state := p.GetState()

	taskState := &TaskState{
		ID:                strconv.Itoa(droplet.ID),
		Name:              fmt.Sprintf("%s-%s", state.PetriTag, definition.Name),
		Definition:        definition,
		TailscaleHostname: fmt.Sprintf("%s-%s", state.PetriTag, definition.Name),
		Status:            provider.TASK_STOPPED,
		ProviderName:      state.Name,
	}

	p.stateMu.Lock()
	p.state.TaskStates[taskState.ID] = taskState
	p.stateMu.Unlock()

	task := &Task{
		state:             taskState,
		removeTask:        p.removeTask,
		logger:            p.logger.With(zap.String("task", definition.Name)),
		doClient:          p.doClient,
		tailscaleSettings: p.tailscaleSettings,
	}

	if err := util.WaitForCondition(ctx, 240*time.Second, 1*time.Second, func() (bool, error) {
		self, err := task.getTailscalePeer(ctx)

		if err != nil {
			return false, nil
		}

		if self == nil {
			return false, nil
		}

		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to wait for tailscale peer in task %s: %w", task.GetState().Name, err)
	}

	ip, err := task.GetIP(ctx)

	if err != nil {
		return nil, err
	}

	task.dockerClient = p.getDockerClientOverride(task.GetState().Name)

	if task.dockerClient == nil {
		task.dockerClient, err = clients.NewDockerClient(ip, p.getDialFunc())

		if err != nil {
			return nil, fmt.Errorf("failed to create docker client: %w", err)
		}
	}

	if err := task.waitForDockerStart(ctx); err != nil {
		return nil, fmt.Errorf("failed to wait for docker start: %w", err)
	}

	_, _, err = task.dockerClient.ImageInspectWithRaw(ctx, definition.Image.Image)
	registryAuth := doConfig["docker_auth"]
	if err != nil {
		p.logger.Info("image not found, pulling", zap.String("image", definition.Image.Image))
		for retries := 5; retries > 0; retries-- {
			err = task.dockerClient.ImagePull(ctx, p.logger, definition.Image.Image, image.PullOptions{
				RegistryAuth: registryAuth,
			})
			if err != nil {
				p.logger.Info("got rate limited on docker pull, sleeping 10 seconds and going again")
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
		if err != nil {
			p.logger.Error("failed to pull image", zap.String("image", definition.Image.Image), zap.Error(err))
			return nil, err
		}
	}

	err = util.WaitForCondition(ctx, 30*time.Second, 1*time.Second, func() (bool, error) {
		_, err := task.dockerClient.ContainerCreate(ctx, &container.Config{
			Image:      definition.Image.Image,
			Entrypoint: definition.Entrypoint,
			Cmd:        definition.Command,
			Tty:        false,
			Hostname:   taskState.Name,
			Labels: map[string]string{
				providerLabelName: state.Name,
				portsLabelName:    strings.Join(definition.Ports, ","),
				nodeNameLabelName: definition.Name,
			},
			Env: convertEnvMapToList(definition.Environment),
		}, &container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: "/docker_volumes",
					Target: definition.DataDir,
				},
			},
			NetworkMode: "host",
		}, nil, nil, taskState.Name)

		if err != nil {
			if client.IsErrConnectionFailed(err) {
				p.logger.Warn("connection failed while creating container, will retry", zap.Error(err))
				return false, nil
			}
			return false, err
		}

		return true, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create container after retries: %w", err)
	}

	return task, nil
}

func (p *Provider) SerializeProvider(context.Context) ([]byte, error) {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	bz, err := json.Marshal(p.state)

	return bz, err
}

func RestoreProvider(ctx context.Context, state []byte, token string, tailscaleSettings TailscaleSettings, opts ...func(*Provider)) (*Provider, error) {
	if token == "" {
		return nil, errors.New("a non-empty token must be passed when restoring a DigitalOcean provider")
	}

	doClient := NewGodoClient(token)
	return RestoreProviderWithClient(ctx, state, doClient, tailscaleSettings, opts...)
}

func RestoreProviderWithClient(_ context.Context, state []byte, doClient DoClient, tailscaleSettings TailscaleSettings, opts ...func(*Provider)) (*Provider, error) {
	if doClient == nil {
		return nil, errors.New("a valid digital ocean client must be passed when restoring the provider")
	}

	var providerState ProviderState

	err := json.Unmarshal(state, &providerState)
	if err != nil {
		return nil, err
	}

	digitalOceanProvider := &Provider{
		state:             &providerState,
		doClient:          doClient,
		tailscaleSettings: tailscaleSettings,
	}

	for _, opt := range opts {
		opt(digitalOceanProvider)
	}

	if digitalOceanProvider.logger == nil {
		digitalOceanProvider.logger = zap.NewNop()
	}

	return digitalOceanProvider, nil
}

func (p *Provider) SerializeTask(ctx context.Context, task provider.TaskI) ([]byte, error) {
	if _, ok := task.(*Task); !ok {
		return nil, fmt.Errorf("task is not a Docker task")
	}

	doTask := task.(*Task)

	bz, err := json.Marshal(doTask.state)

	if err != nil {
		return nil, err
	}

	return bz, nil
}

func (p *Provider) DeserializeTask(ctx context.Context, bz []byte) (provider.TaskI, error) {
	var taskState TaskState

	err := json.Unmarshal(bz, &taskState)
	if err != nil {
		return nil, err
	}

	task := &Task{
		state:      &taskState,
		removeTask: p.removeTask,
	}

	if err := p.initializeDeserializedTask(task); err != nil {
		return nil, err
	}

	return task, nil
}

func (p *Provider) initializeDeserializedTask(task *Task) error {
	taskState := task.GetState()
	task.logger = p.logger.With(zap.String("task", taskState.Name))
	task.doClient = p.doClient
	task.dockerClient = p.getDockerClientOverride(task.GetState().Name)
	task.tailscaleSettings = p.tailscaleSettings

	if task.dockerClient == nil {
		ip, err := task.GetIP(context.Background())
		if err != nil {
			return err
		}

		task.dockerClient, err = clients.NewDockerClient(ip, p.getDialFunc())
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) Teardown(ctx context.Context) error {
	p.logger.Info("tearing down DigitalOcean provider")

	if err := p.teardownTasks(ctx); err != nil {
		return err
	}

	if err := p.teardownFirewall(ctx); err != nil {
		return err
	}

	if err := p.teardownTag(ctx); err != nil {
		return err
	}

	if err := p.teardownDomains(ctx); err != nil {
		return err
	}

	return nil
}

func (p *Provider) teardownTasks(ctx context.Context) error {
	return p.doClient.DeleteDropletByTag(ctx, p.GetState().PetriTag)
}

func (p *Provider) teardownFirewall(ctx context.Context) error {
	_, err := p.doClient.GetFirewall(ctx, p.GetState().FirewallID)

	// firewall is already deleted at this point or we have bad state (nothing we can do in this case)
	if errors.Is(err, ErrorResourceNotFound) {
		return nil
	}

	return p.doClient.DeleteFirewall(ctx, p.GetState().FirewallID)
}

func (p *Provider) teardownTag(ctx context.Context) error {
	return p.doClient.DeleteTag(ctx, p.GetState().PetriTag)
}

func (p *Provider) teardownDomains(ctx context.Context) error {
	if p.domain == "" {
		return nil
	}

	var multiErr error

	for _, id := range p.state.DomainIDs {
		err := p.doClient.DeleteDomain(ctx, p.domain, id)
		if err != nil {
			multiErr = errors.Join(err, fmt.Errorf("failed to delete domain %d: %w", id, err))
			continue
		}
	}

	return multiErr
}

func (p *Provider) removeTask(_ context.Context, taskID string) error {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	delete(p.state.TaskStates, taskID)

	return nil
}

func (p *Provider) createTag(ctx context.Context, tagName string) (*godo.Tag, error) {
	req := &godo.TagCreateRequest{
		Name: tagName,
	}

	return p.doClient.CreateTag(ctx, req)
}

func (p *Provider) CreateDomains(ctx context.Context, domains map[string]string) error {
	if p.domain == "" {
		return nil
	}

	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	for domain, ip := range domains {
		p.logger.Info("creating domain", zap.String("name", domain), zap.String("ip", ip))

		req := &godo.DomainRecordEditRequest{
			Type: "A",
			Name: domain,
			Data: ip,
			TTL:  300,
		}

		resp, err := p.doClient.CreateDomain(ctx, p.domain, req)
		if err != nil {
			return fmt.Errorf("failed to create domain %s: %w", domain, err)
		}

		p.state.DomainIDs = append(p.state.DomainIDs, resp.ID)
	}

	return nil
}

func (p *Provider) GetState() ProviderState {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()
	return *p.state
}

func (p *Provider) getDialFunc() func(ctx context.Context, network, address string) (net.Conn, error) {
	return p.tailscaleSettings.Server.Dial
}

func (p *Provider) getDockerClientOverride(task string) clients.DockerClient {
	p.stateMu.Lock()
	defer p.stateMu.Unlock()

	if dockerClient, ok := p.dockerClientOverrides[task]; ok {
		return dockerClient
	}

	return nil
}

func (p *Provider) GetType() string {
	return types.DigitalOcean
}

func (p *Provider) GetName() string {
	return p.state.Name
}
