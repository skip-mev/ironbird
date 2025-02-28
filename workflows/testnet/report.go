package testnet

import (
	"bytes"
	"fmt"
	"time"

	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/activities/loadtest"
	"github.com/skip-mev/ironbird/activities/testnet"
	"github.com/skip-mev/ironbird/util"
	"go.temporal.io/sdk/workflow"
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
	loadTestResults  *loadtest.LoadTestResult
	loadTestStatus   string
	loadTestConfig   string
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

func (r *Report) UpdateLoadTest(ctx workflow.Context, status string, config string, results *loadtest.LoadTestResult) error {
	r.loadTestStatus = status
	r.loadTestConfig = config
	r.loadTestResults = results
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

func (r *Report) addLoadTestResultsToMarkdown(md *markdown.Markdown) {
	if r.loadTestStatus == "" && r.loadTestResults == nil {
		return
	}

	md.HorizontalRule()
	md.H1("Load Test")

	// Show status and config first
	if r.loadTestStatus != "" {
		md.H2("Status")
		md.PlainText(r.loadTestStatus)
		if r.loadTestConfig != "" {
			md.H2("Configuration")
			md.PlainText(r.loadTestConfig)
		}
	}

	// If we have results and no error, show them
	if r.loadTestResults != nil && r.loadTestResults.Error == "" {
		// Overall stats
		md.H2("🎯 Overall Statistics")
		rows := [][]string{
			{"Total Transactions", fmt.Sprintf("%d", r.loadTestResults.Overall.TotalTransactions)},
			{"Successful Transactions", fmt.Sprintf("%d", r.loadTestResults.Overall.SuccessfulTransactions)},
			{"Failed Transactions", fmt.Sprintf("%d", r.loadTestResults.Overall.FailedTransactions)},
			{"Average Gas Per Transaction", fmt.Sprintf("%d", r.loadTestResults.Overall.AvgGasPerTransaction)},
			{"Average Block Gas Utilization", fmt.Sprintf("%.2f%%", r.loadTestResults.Overall.AvgBlockGasUtilization*100)},
			{"Runtime", r.loadTestResults.Overall.Runtime.String()},
			{"Blocks Processed", fmt.Sprintf("%d", r.loadTestResults.Overall.BlocksProcessed)},
		}
		md.Table(markdown.TableSet{
			Header: []string{"Metric", "Value"},
			Rows:   rows,
		})

		// Message type stats
		md.H2("📊 Message Type Statistics")
		for msgType, stats := range r.loadTestResults.ByMessage {
			md.H3(string(msgType))
			rows := [][]string{
				{"Total Transactions", fmt.Sprintf("%d", stats.Transactions.Total)},
				{"Successful Transactions", fmt.Sprintf("%d", stats.Transactions.Successful)},
				{"Failed Transactions", fmt.Sprintf("%d", stats.Transactions.Failed)},
				{"Average Gas", fmt.Sprintf("%d", stats.Gas.Average)},
				{"Min Gas", fmt.Sprintf("%d", stats.Gas.Min)},
				{"Max Gas", fmt.Sprintf("%d", stats.Gas.Max)},
				{"Total Gas", fmt.Sprintf("%d", stats.Gas.Total)},
			}
			md.Table(markdown.TableSet{
				Header: []string{"Metric", "Value"},
				Rows:   rows,
			})

			if len(stats.Errors.BroadcastErrors) > 0 {
				md.H4("Errors")
				for errType, count := range stats.Errors.ErrorCounts {
					md.PlainText(fmt.Sprintf("- %s: %d occurrences\n", errType, count))
				}
			}
		}

		// Node stats
		md.H2("🖥️ Node Statistics")
		for nodeAddr, stats := range r.loadTestResults.ByNode {
			md.H3(nodeAddr)
			rows := [][]string{
				{"Total Transactions", fmt.Sprintf("%d", stats.TransactionStats.Total)},
				{"Successful Transactions", fmt.Sprintf("%d", stats.TransactionStats.Successful)},
				{"Failed Transactions", fmt.Sprintf("%d", stats.TransactionStats.Failed)},
				{"Average Gas", fmt.Sprintf("%d", stats.GasStats.Average)},
				{"Min Gas", fmt.Sprintf("%d", stats.GasStats.Min)},
				{"Max Gas", fmt.Sprintf("%d", stats.GasStats.Max)},
			}
			md.Table(markdown.TableSet{
				Header: []string{"Metric", "Value"},
				Rows:   rows,
			})

			md.H4("Message Distribution")
			var msgRows [][]string
			for msgType, count := range stats.MessageCounts {
				msgRows = append(msgRows, []string{string(msgType), fmt.Sprintf("%d", count)})
			}
			md.Table(markdown.TableSet{
				Header: []string{"Message Type", "Count"},
				Rows:   msgRows,
			})
		}

		// Block stats
		md.H2("📦 Block Statistics Summary")
		if len(r.loadTestResults.ByBlock) > 0 {
			var totalGasUtilization float64
			var maxGasUtilization float64
			minGasUtilization := r.loadTestResults.ByBlock[0].GasUtilization
			var maxGasBlock int64
			var minGasBlock int64

			for _, block := range r.loadTestResults.ByBlock {
				totalGasUtilization += block.GasUtilization
				if block.GasUtilization > maxGasUtilization {
					maxGasUtilization = block.GasUtilization
					maxGasBlock = block.BlockHeight
				}
				if block.GasUtilization < minGasUtilization {
					minGasUtilization = block.GasUtilization
					minGasBlock = block.BlockHeight
				}
			}

			avgGasUtilization := totalGasUtilization / float64(len(r.loadTestResults.ByBlock))
			rows := [][]string{
				{"Total Blocks", fmt.Sprintf("%d", len(r.loadTestResults.ByBlock))},
				{"Average Gas Utilization", fmt.Sprintf("%.2f%%", avgGasUtilization*100)},
				{"Min Gas Utilization", fmt.Sprintf("%.2f%% (Block %d)", minGasUtilization*100, minGasBlock)},
				{"Max Gas Utilization", fmt.Sprintf("%.2f%% (Block %d)", maxGasUtilization*100, maxGasBlock)},
			}
			md.Table(markdown.TableSet{
				Header: []string{"Metric", "Value"},
				Rows:   rows,
			})
		}
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

	if r.loadTestStatus != "" || r.loadTestResults != nil {
		r.addLoadTestResultsToMarkdown(md)
	}

	if err := md.Build(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
