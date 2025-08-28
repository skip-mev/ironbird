## Monitoring your tasks with the monitoring package (batteries included)

The `monitoring` package in `petri/core` allows you to easily set up Prometheus and Grafana to ingest metrics
from your tasks.

### Setting up Prometheus

The monitoring package has a function `SetupPrometheusTask` that handles the setting up and configuration of a
Prometheus task.

```go
prometheusTask, err := monitoring.SetupPrometheusTask(ctx, logger, provider, monitoring.PrometheusOptions{
	Targets: endpoints, // these endpoints are in the format host:port, make sure they are reachable from the Prometheus task
	ProviderSpecificConfig: struct{}{} // if any apply
})

if err != nil {
	panic(err)
}

err = prometheusTask.Start(ctx, false)

if err != nil {
	panic(err)
}
```

### Setting up Prometheus for your chain

The set up for ingesting metrics from the chain nodes is pretty similar, with one caveat of getting the node
metrics endpoints. The Prometheus task has to be created after the chain is started, as only after that
you can be sure that you can receive the correct external IP for the nodes.

```go
var endpoints []string

for _, node := range append(chain.GetValidators(), chain.GetNodes()...) {
	endpoint, err := node.GetTask().GetIP(ctx)
	if err != nil {
		panic(err)
	}

	endpoints = append(endpoints, fmt.Sprintf("%s:26660", endpoint))
}
```

### Setting up Grafana

The monitoring package has a similar function for setting up a Grafana task called `SetupGrafanaTask`.
The difference between Prometheus and Grafana setup is that Grafana additionally requires a snapshot of a Grafana dashboard
exported in JSON. 

You can look up how to export a dashboard in JSON format [here](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/view-dashboard-json-model/).
You need to note down the dashboard ID if you want to take a snapshot and replace the `version` in the JSON model with `1`.
Additionally, you need to replace all mentions of your Prometheus data source with `petri_prometheus`.

```go
import _ "embed"

//go:embed files/dashboard.json
var dashboardJSON string

grafanaTask, err := monitoring.SetupGrafanaTask(ctx, logger, provider, monitoring.GrafanaOptions{
    DashboardJSON: dashboardJSON,
    PrometheusURL: fmt.Sprintf("http://%s", prometheusIP),
    ProviderSpecificConfig: struct{}{} // if any apply
})

if err != nil {
    panic(err)
}

err = grafanaTask.Start(ctx, false)

if err != nil {
    panic(err)
}

grafanaIP, err := s.grafanaTask.GetExternalAddress(ctx, "3000")

if err != nil {
    panic(err)
}

fmt.Printf("Visit Grafana at http://%s\n", grafanaIP)
```

### Taking a public snapshot of a Grafana dashboard

You can take a public snapshot of a Grafana dashboard by using the `TakeSnapshot` function.

```go
snapshotURL, err := monitoring.TakeSnapshot(ctx, "<dashboard_uid>", "<grafana_ip>")

if err != nil {
    panic(err)
}

fmt.Printf("Visit the snapshot at %s\n", snapshotURL)
```
