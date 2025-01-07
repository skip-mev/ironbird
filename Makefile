temporal-reset:
	temporal workflow list -o json | jq -r '.[] | select (.status == "WORKFLOW_EXECUTION_STATUS_RUNNING") | .execution.workflowId' | xargs -I{} temporal workflow terminate --reason lol --workflow-id "{}"

do-reset:
	doctl compute droplet list | grep petri-droplet | cut -d' ' -f1 | xargs -I{} doctl compute droplet delete -f {} && doctl compute firewall list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute firewall delete -f {} && doctl compute ssh-key list | grep petri | cut -d' ' -f1 | xargs -I{} doctl compute ssh-key delete -f {}

reset: do-reset temporal-reset

.PHONY: reset temporal-reset do-reset

