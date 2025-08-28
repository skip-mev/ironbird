package apps

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/skip-mev/ironbird/petri/core/provider"
	"github.com/skip-mev/ironbird/petri/core/provider/digitalocean"
)

type LoadBalancerDomain struct {
	Domain   string   `json:"domain"`
	IPs      []string `json:"ips,omitempty"`
	Protocol string   `json:"protocol"`
}

func (lbd LoadBalancerDomain) Validate() error {
	if lbd.Domain == "" {
		return fmt.Errorf("domain must be specified")
	}

	if len(lbd.IPs) == 0 {
		return fmt.Errorf("at least one IP must be specified")
	}

	if lbd.Protocol != "http" && lbd.Protocol != "grpc" {
		return fmt.Errorf("protocol must be either 'http' or 'grpc'")
	}
	return nil
}

type LoadBalancerDefinition struct {
	DigitalOceanConfig map[string]string
	Domains            []LoadBalancerDomain
	SSLCertificate     []byte
	SSLKey             []byte
}

func (lbd LoadBalancerDefinition) Validate() error {
	if len(lbd.Domains) == 0 {
		return fmt.Errorf("at least one domain must be specified")
	}

	if len(lbd.SSLKey) == 0 && len(lbd.SSLCertificate) > 0 {
		return fmt.Errorf("SSL key must be provided if SSL certificate is specified")
	}

	if len(lbd.SSLCertificate) == 0 && len(lbd.SSLKey) > 0 {
		return fmt.Errorf("SSL certificate must be provided if SSL key is specified")
	}

	var err error

	for _, domain := range lbd.Domains {
		err = errors.Join(domain.Validate())
	}

	return err
}

// CaddyTLSTemplate is used for TLS termination
const CaddyTlsTemplate = "tls /caddy/cert.pem /caddy/key.pem"

// CaddyHttpDomainTemplate is used for HTTP services
// it uses the default HTTP transport and sends traffic
// to the second argument in the template
const CaddyHttpDomainTemplate = `%s {
	log
	
	handle {
		reverse_proxy %s 
	}

	%s
}
`

// CaddyGrpcDomainTemplate is used for gRPC services -
// it uses h2c (HTTP/2 cleartext) for the transport
// and requires TLS termination, assuming that the endpoint
// is cleartext
const CaddyGrpcDomainTemplate = `%s {
	log

	handle {
		reverse_proxy %s {
			transport http {
				# Use HTTP/2 cleartext for gRPC
				versions h2c
			}
		}
	}

	%s
}
`

// LaunchLoadBalancer only supports the DigitalOcean provider
func LaunchLoadBalancer(ctx context.Context, p *digitalocean.Provider, rootDomain string, definition LoadBalancerDefinition) (provider.TaskI, error) {
	task, err := p.CreateTask(ctx, provider.TaskDefinition{
		Name: "loadbalancer",
		Image: provider.ImageDefinition{
			Image: "caddy:2-alpine",
			UID:   "0",
			GID:   "0",
		},
		Ports:      []string{},
		DataDir:    "/caddy",
		Entrypoint: []string{"caddy", "run", "--config", "/caddy/Caddyfile"},

		ProviderSpecificConfig: definition.DigitalOceanConfig,
	})

	if err != nil {
		return nil, err
	}

	tlsTemplate := ""

	if definition.SSLCertificate != nil && definition.SSLKey != nil {
		if err := task.WriteFile(ctx, "cert.pem", definition.SSLCertificate); err != nil {
			return nil, err
		}

		if err := task.WriteFile(ctx, "key.pem", definition.SSLKey); err != nil {
			return nil, err
		}

		tlsTemplate = CaddyTlsTemplate
	}

	caddyConfig := ""

	for _, domain := range definition.Domains {
		fullDomain := fmt.Sprintf("%s.%s", domain.Domain, rootDomain)
		var template string
		if domain.Protocol == "http" {
			template = CaddyHttpDomainTemplate
		} else if domain.Protocol == "grpc" {
			template = CaddyGrpcDomainTemplate
		}

		ipDirective := strings.Join(domain.IPs, " ")
		caddyConfig += fmt.Sprintf(template, fullDomain, ipDirective, tlsTemplate)
	}

	if err := task.WriteFile(ctx, "Caddyfile", []byte(caddyConfig)); err != nil {
		return nil, err
	}

	if err := task.Start(ctx); err != nil {
		return nil, err
	}

	// hack: until we figure out how to best handle returning addresses in providers
	doTask, ok := task.(*digitalocean.Task)
	if !ok {
		return nil, fmt.Errorf("task is not a DigitalOcean task")
	}

	droplet, err := doTask.GetDroplet(ctx)

	if err != nil {
		return nil, err
	}

	ip, err := droplet.PublicIPv4()

	if err != nil {
		return nil, err
	}

	domains := make(map[string]string, len(definition.Domains))

	for _, domain := range definition.Domains {
		domains[domain.Domain] = ip
	}

	if err := p.CreateDomains(ctx, domains); err != nil {
		return nil, err
	}

	return task, nil
}
