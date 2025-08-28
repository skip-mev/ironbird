# Petri providers

In Petri, providers are meant to abstract the implementation details
of creating workloads (containers) from the downstream dependencies.
At the current state, Petri supports two providers: `docker` and `digitalocean`.

In most cases, you won't interact with the providers directly, but rather use the 
`chain`, `node`, etc. packages to create a chain. If you still want to use the `provider`
package directly to create custom workloads, feel free to continue reading this doc.

## Using a provider

Instead of interacting with providers implementing the `Provider` interface directly,
you should use the `NewTask` function to create tasks. Tasks are a higher level
abstraction that allows you to create workloads in a more declarative way.

A working example on how to create a task:
```go
package main

import (
	"context"
	"fmt"
	"github.com/skip-mev/petri/general/v2/provider"
	"github.com/skip-mev/petri/general/v2/provider/digitalocean"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	doProvider, err := docker.NewDockerProvider(context.Background(), logger, "petri_docker")
	if err != nil {
		panic(err)
	}

	task, err := provider.CreateTask(context.Background(), logger, doProvider, provider.TaskDefinition{
		Name: "petri_example",
		Image: provider.ImageDefinition{
			Image: "nginx",
		},
		DataDir: "/test",
	})

	if err != nil {
		panic(err)
	}

	err = task.Start(context.Background(), false)

	if err != nil {
		panic(err)
	}
}
```

## Interacting with tasks

Once you have a task, you can interact with it using the following methods:
- `Start`: starts the task and its sidecars
- `Stop`: stops the task and its sidecars
- `(Write/Read)File` : writes/reads a file from the container
- `RunCommand`: runs a command inside the container
- `GetExternalAddress`: given a port, it returns the external address 
                        of the container for that specific port
                        (useful for API-based interaction)

## Hooking into the lifecycle of a task

You can hook into the lifecycle of a task or its sidecars (since they're also a Task)
by using the `SetPreStart` / `SetPostStop` methods. By default, a Task does not have
any pre-start or post-stop hooks.

Example of using a pre-start hook:

```go
package main

package main

import (
	"context"
	"fmt"
	"github.com/skip-mev/petri/general/v2/provider"
	"github.com/skip-mev/petri/general/v2/provider/digitalocean"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
	doProvider, err := docker.NewDockerProvider(context.Background(), logger, "petri_docker")
	if err != nil {
		panic(err)
	}

	task, err := provider.CreateTask(context.Background(), logger, doProvider, provider.TaskDefinition{
		Name: "petri_example",
		Image: provider.ImageDefinition{
			Image: "nginx",
		},
		DataDir: "/test",
	})

	if err != nil {
		panic(err)
	}

	err = task.Start(context.Background(), false)

	if err != nil {
		panic(err)
	}
	
    task.SetPreStart(func(ctx context.Context, task *provider.Task) error {
		_, _, _, err := task.RunCommand(ctx, []string{"mv",  "/mnt/index.html", "/usr/share/nginx/html"})
		
        return err
    })
}
```

## Provider specific configuration

Every task can have its own provider-specific configuration.
The provider-specific configuration has a type of `interface{}` and each provider
is responsible for verifying that the configuration is of the correct type.