package digitalocean

import (
	"context"
	"errors"
	"fmt"

	"github.com/digitalocean/godo"
)

// DoClient defines the interface for DigitalOcean API operations used by the provider
type DoClient interface {
	// Droplet operations
	CreateDroplet(ctx context.Context, req *godo.DropletCreateRequest) (*godo.Droplet, error)
	GetDroplet(ctx context.Context, dropletID int) (*godo.Droplet, error)
	ListDroplets(ctx context.Context, opts *godo.ListOptions) ([]godo.Droplet, error)
	DeleteDropletByTag(ctx context.Context, tag string) error
	DeleteDropletByID(ctx context.Context, id int) error

	// Firewall operations
	CreateFirewall(ctx context.Context, req *godo.FirewallRequest) (*godo.Firewall, error)
	GetFirewall(ctx context.Context, firewallID string) (*godo.Firewall, error)
	ListFirewalls(ctx context.Context, opts *godo.ListOptions) ([]godo.Firewall, error)
	DeleteFirewall(ctx context.Context, firewallID string) error

	// Domain operations
	CreateDomain(ctx context.Context, rootDomain string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, error)
	GetDomain(ctx context.Context, rootDomain string, recordId int) (*godo.DomainRecord, error)
	DeleteDomain(ctx context.Context, rootDomain string, recordId int) error

	// SSH Key operations
	CreateKey(ctx context.Context, req *godo.KeyCreateRequest) (*godo.Key, error)
	DeleteKeyByFingerprint(ctx context.Context, fingerprint string) error
	GetKeyByFingerprint(ctx context.Context, fingerprint string) (*godo.Key, error)

	// Tag operations
	CreateTag(ctx context.Context, req *godo.TagCreateRequest) (*godo.Tag, error)
	ListTags(ctx context.Context, opts *godo.ListOptions) ([]godo.Tag, error)
	DeleteTag(ctx context.Context, tag string) error
}

// godoClient implements the DoClient interface using the actual godo.Client
type godoClient struct {
	*godo.Client
}

var (
	ErrorResourceNotFound = errors.New("resource not found")
	ErrorEmptyResponse    = errors.New("unexpected empty response")
)

func NewGodoClient(token string) DoClient {
	return &godoClient{Client: godo.NewFromToken(token)}
}

func checkResponse(res *godo.Response, err error) error {
	if err != nil {
		return err
	}

	if res == nil {
		return ErrorEmptyResponse
	}

	if res.StatusCode == 404 {
		return ErrorResourceNotFound
	}

	if res.StatusCode > 299 || res.StatusCode < 200 {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	return nil
}

// Droplet operations
func (c *godoClient) CreateDroplet(ctx context.Context, req *godo.DropletCreateRequest) (*godo.Droplet, error) {
	droplet, res, err := c.Droplets.Create(ctx, req)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return droplet, nil
}

func (c *godoClient) GetDroplet(ctx context.Context, dropletID int) (*godo.Droplet, error) {
	droplet, res, err := c.Droplets.Get(ctx, dropletID)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return droplet, nil
}

func (c *godoClient) ListDroplets(ctx context.Context, opts *godo.ListOptions) ([]godo.Droplet, error) {
	droplets, res, err := c.Droplets.List(ctx, opts)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return droplets, nil
}

func (c *godoClient) DeleteDropletByTag(ctx context.Context, tag string) error {
	res, err := c.Droplets.DeleteByTag(ctx, tag)
	return checkResponse(res, err)
}

func (c *godoClient) DeleteDropletByID(ctx context.Context, id int) error {
	res, err := c.Droplets.Delete(ctx, id)
	return checkResponse(res, err)
}

// Firewall operations
func (c *godoClient) CreateFirewall(ctx context.Context, req *godo.FirewallRequest) (*godo.Firewall, error) {
	firewall, res, err := c.Firewalls.Create(ctx, req)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return firewall, nil
}

func (c *godoClient) ListFirewalls(ctx context.Context, opts *godo.ListOptions) ([]godo.Firewall, error) {
	firewalls, res, err := c.Firewalls.List(ctx, opts)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return firewalls, nil
}

func (c *godoClient) DeleteFirewall(ctx context.Context, firewallID string) error {
	res, err := c.Firewalls.Delete(ctx, firewallID)
	return checkResponse(res, err)
}

// Returns nil in case the firewall does not exist anymore
func (c *godoClient) GetFirewall(ctx context.Context, firewallID string) (*godo.Firewall, error) {
	firewall, res, err := c.Firewalls.Get(ctx, firewallID)

	if err := checkResponse(res, err); err != nil {
		return nil, err
	}

	return firewall, nil
}

func (c *godoClient) CreateDomain(ctx context.Context, rootDomain string, req *godo.DomainRecordEditRequest) (*godo.DomainRecord, error) {
	domain, res, err := c.Domains.CreateRecord(ctx, rootDomain, req)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return domain, nil
}

func (c *godoClient) GetDomain(ctx context.Context, rootDomain string, recordId int) (*godo.DomainRecord, error) {
	domainResp, res, err := c.Domains.Record(ctx, rootDomain, recordId)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return domainResp, nil
}

func (c *godoClient) DeleteDomain(ctx context.Context, rootDomain string, recordId int) error {
	res, err := c.Domains.DeleteRecord(ctx, rootDomain, recordId)
	return checkResponse(res, err)
}

// SSH Key operations
func (c *godoClient) CreateKey(ctx context.Context, req *godo.KeyCreateRequest) (*godo.Key, error) {
	key, res, err := c.Keys.Create(ctx, req)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return key, nil
}

func (c *godoClient) DeleteKeyByFingerprint(ctx context.Context, fingerprint string) error {
	res, err := c.Keys.DeleteByFingerprint(ctx, fingerprint)
	return checkResponse(res, err)
}

func (c *godoClient) GetKeyByFingerprint(ctx context.Context, fingerprint string) (*godo.Key, error) {
	key, res, err := c.Keys.GetByFingerprint(ctx, fingerprint)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return key, nil
}

// Tag operations
func (c *godoClient) CreateTag(ctx context.Context, req *godo.TagCreateRequest) (*godo.Tag, error) {
	tag, res, err := c.Tags.Create(ctx, req)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return tag, nil
}

func (c *godoClient) ListTags(ctx context.Context, opts *godo.ListOptions) ([]godo.Tag, error) {
	tags, res, err := c.Tags.List(ctx, opts)
	if err := checkResponse(res, err); err != nil {
		return nil, err
	}
	return tags, nil
}

func (c *godoClient) DeleteTag(ctx context.Context, tag string) error {
	res, err := c.Tags.Delete(ctx, tag)
	return checkResponse(res, err)
}
