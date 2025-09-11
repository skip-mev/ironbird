package provider

import (
	"context"
	"net"
	"strings"
)

// TaskStatus defines the status of a task's underlying workload
type TaskStatus int

// RemoveTaskFunc is a callback function type for removing a task from its provider
type RemoveTaskFunc func(ctx context.Context, taskID string) error

const (
	TASK_STATUS_UNDEFINED TaskStatus = iota
	TASK_RUNNING
	TASK_STOPPED
	TASK_PAUSED
	TASK_RESTARTING
)

type TaskI interface {
	Start(context.Context) error
	Stop(context.Context) error
	Destroy(context.Context) error

	GetDefinition() TaskDefinition

	GetStatus(context.Context) (TaskStatus, error)

	Modify(context.Context, TaskDefinition) error

	WriteFile(context.Context, string, []byte) error
	ReadFile(context.Context, string) ([]byte, error)
	DownloadDir(context.Context, string, string) error

	GetIP(context.Context) (string, error)
	GetExternalAddress(context.Context, string) (string, error)
	DialContext() func(context.Context, string, string) (net.Conn, error)

	RunCommand(context.Context, []string) (string, string, int, error)
}

type ProviderI interface {
	CreateTask(context.Context, TaskDefinition) (TaskI, error) // should this create a TaskI or the resources behind it?
	SerializeTask(context.Context, TaskI) ([]byte, error)
	DeserializeTask(context.Context, []byte) (TaskI, error)

	Teardown(context.Context) error

	SerializeProvider(context.Context) ([]byte, error)
	GetType() string
	GetName() string
}

// IsECRImage checks if an image name is from Amazon ECR
func IsECRImage(imageName string) bool {
	return strings.Contains(imageName, ".amazonaws.com")
}
