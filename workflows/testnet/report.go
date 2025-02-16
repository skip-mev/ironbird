package testnet

import (
	"bytes"
	"fmt"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/util"
	"go.temporal.io/sdk/workflow"
	"time"
)

type Report struct {
	workflowOptions *WorkflowOptions
	start           time.Time
	checkId         int64
	name            string
	status          string
	title           string
	summary         string
	conclusion      string

	nodes            []testnet.Node
	observabilityURL string
	screenshots      map[string]string
}

func NewReport(ctx workflow.Context, name, title, summary string, opts *WorkflowOptions) (*Report, error) {
	if opts == nil {
		return nil, fmt.Errorf("workflow options are required")
	}

	report := &Report{
		workflowOptions: opts,
		start:           workflow.Now(ctx),
		status:          "queued",
		name:            name,
		title:           title,
		summary:         summary,
	}

	checkId, err := report.CreateCheck(ctx)

	if err != nil {
		return nil, err
	}

	report.checkId = checkId

	return report, nil
}

func (r *Report) CreateCheck(ctx workflow.Context) (int64, error) {
	options := r.workflowOptions.GenerateCheckOptions(
		r.name,
		r.status,
		r.title,
		r.summary,
		"",
		nil,
	)

	var checkId int64

	if err := workflow.ExecuteActivity(ctx, githubActivities.CreateCheck, options).Get(ctx, &checkId); err != nil {
		return -1, err
	}

	return checkId, nil
}

func (r *Report) UpdateCheck(ctx workflow.Context) error {
	output, err := r.Markdown()

	if err != nil {
		return err
	}

	var conclusion *string

	if r.conclusion != "" {
		conclusion = util.StringPtr(r.conclusion)
	}

	options := r.workflowOptions.GenerateCheckOptions(
		r.name,
		r.status,
		r.title,
		r.summary,
		output,
		conclusion,
	)

	return workflow.ExecuteActivity(ctx, githubActivities.UpdateCheck, r.checkId, options).Get(ctx, nil)
}

func (r *Report) TimeSinceStart(ctx workflow.Context) time.Duration {
	return workflow.Now(ctx).Sub(r.start)
}

func (r *Report) Conclude(ctx workflow.Context, status, conclusion, title string) error {
	r.status = status
	r.conclusion = conclusion
	r.title = title
	r.summary = fmt.Sprintf("The job took %s to complete", r.TimeSinceStart(ctx).String())

	return r.UpdateCheck(ctx)
}

func (r *Report) SetStatus(ctx workflow.Context, status, title, summary string) error {
	r.status = status
	r.title = title
	r.summary = summary

	return r.UpdateCheck(ctx)
}

func (r *Report) SetScreenshots(ctx workflow.Context, screenshots map[string]string) error {
	r.screenshots = screenshots
	return r.UpdateCheck(ctx)
}

func (r *Report) SetNodes(ctx workflow.Context, nodes []testnet.Node) error {
	r.nodes = nodes
	return r.UpdateCheck(ctx)
}

func (r *Report) SetObservabilityURL(ctx workflow.Context, url string) error {
	r.observabilityURL = url
	return r.UpdateCheck(ctx)
}

func (r *Report) addNodesToMarkdown(md *markdown.Markdown) {
	rows := make([][]string, len(r.nodes))

	for i, node := range r.nodes {
		rows[i] = []string{node.Name, node.Rpc, node.Lcd}
	}

	md.HorizontalRule()
	md.H1("Nodes")
	md.Table(markdown.TableSet{
		Header: []string{"Name", "RPC", "LCD"},
		Rows:   rows,
	})
}

func (r *Report) addScreenshotsToMarkdown(md *markdown.Markdown) {
	md.HorizontalRule()
	md.H1("Observability graphs")

	for name, url := range r.screenshots {
		md.HorizontalRule()
		md.H3(fmt.Sprintf("Screenshot - %s", name))
		md.PlainText(fmt.Sprintf("![](%s)", url))
	}
}

func (r *Report) Markdown() (string, error) {
	var buf bytes.Buffer

	md := markdown.NewMarkdown(&buf)

	if len(r.nodes) > 0 {
		r.addNodesToMarkdown(md)
	}

	if r.observabilityURL != "" {
		md.HorizontalRule()
		md.H1("Observability")
		md.PlainText(fmt.Sprintf("Grafana: [%s](%s)", r.observabilityURL, r.observabilityURL))
	}

	if len(r.screenshots) > 0 {
		r.addScreenshotsToMarkdown(md)
	}

	if err := md.Build(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
