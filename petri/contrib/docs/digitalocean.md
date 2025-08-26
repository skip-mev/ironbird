## DigitalOcean provider setup

The DigitalOcean provider is a provider that uses the DigitalOcean API to create and manage droplets.
It's different from the Docker provider in that it requires a one-time set-up.

Petri includes a Packer definition file for an Ubuntu image that already has Docker set up and remotely exposed
to the world. This is done for optimization reasons - creating an image one time is much faster than installing
Docker and other dependencies on every created instance (1 minute vs >5 minutes).

This only needs to be done once on your DigitalOcean account. After that, you can use the DigitalOcean provider
as you would use the Docker provider.

### Prerequisites

- A DigitalOcean API token
- [Packer](https://developer.hashicorp.com/packer/tutorials/docker-get-started/get-started-install-cli) 

### Creating the Packer image

1. Rename the `contrib/digitalocean/petri_docker.pkr.hcl.example` file to `contrib/digitalocean/petri_docker.pkr.hcl`
2. Replace `<DO_API_TOKEN>` with your DigitalOcean API token 
3. Replace `<GRAF_PYRO_USER>` and `<GRAF_PYRO_PASS>` in `config.alloy` with your Grafana username and password for Pyroscope profiling data publication.
4. Include the regions you're going to run Petri on in the "snapshot_regions" variable.
5. Run `packer build petri_docker.pkr.hcl`

### Finding the image ID of your snapshot

You can use the `doctl` (DigitalOcean CLI tool) to find the image ID of your snapshot.

```bash
doctl compute snapshot list
```

Denote the ID of the `petri-ubuntu-xxx` image.

### Using the Petri Docker image

**Using a provider directly**

When using a provider directly to create a task, you can just modify the `TaskDefinition` to include a DigitalOcean
specific configuration that includes the image ID.

```go
provider.TaskDefinition {
		Name: "petri_example",
		Image: provider.ImageDefinition{
			Image: "nginx",
		},
		DataDir: "/test",
		ProviderSpecific: provider.ProviderSpecific{
			"image_id": <IMAGE_ID>,
		},
}
```

**Using a Chain**

When using a Chain to create nodes, you have to include a `NodeDefinitionModifier` in the `ChainConfig` that
includes the Image ID in the provider specific configuration

```go
spec = petritypes.ChainConfig{
	// other configuration removed
	NodeDefinitionModifier: func(def provider.TaskDefinition, nodeConfig petritypes.NodeConfig) provider.TaskDefinition {
		doConfig := digitalocean.DigitalOceanTaskConfig{
			Size: "s-1vcpu-2gb",
			Region: "ams3", // only choose regions that the snapshot is available in
			ImageID: <IMAGE_ID>,
		}

		def.ProviderSpecificConfig = doConfig

		return def
	},
}
```
