package pipeline

import (
	"context"
	"github.com/skip-mev/petri/core/v2/provider"
	"go.uber.org/zap"
)

func (n *NodeActivity) MonitorContainer(ctx context.Context, name, id string) (string, error) {
	p, err := n.ProviderCreator(ctx, zap.NewNop(), name)

	if err != nil {
		return "", err
	}

	ts, err := p.GetTaskStatus(ctx, id)

	if err != nil {
		return "", err
	}

	switch ts {
	case provider.TASK_RUNNING:
		return "running", nil
	case provider.TASK_STOPPED:
		return "stopped", nil
	case provider.TASK_PAUSED:
		return "paused", nil
	case provider.TASK_STATUS_UNDEFINED:
	default:
		return "unknown", nil
	}

	return "unknown", nil
}
