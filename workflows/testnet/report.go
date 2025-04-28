package testnet

import (
	"bytes"
	"fmt"
	catalysttypes "github.com/skip-mev/catalyst/pkg/types"
	"github.com/skip-mev/ironbird/activities/github"
	"github.com/skip-mev/ironbird/messages"
	"github.com/skip-mev/ironbird/types"
	"time"

	"github.com/nao1215/markdown"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/ironbird/util"
	"go.temporal.io/sdk/workflow"
)

type Report struct {
	workflowRequest messages.TestnetWorkflowRequest

	start      time.Time
	checkId    int64
	name       string
	status     string
	title      string
	summary    string
	conclusion string

	nodes           []testnettypes.Node
	buildResult     messages.BuildDockerImageResponse
	dashboards      map[string]string
	loadTestResults *catalysttypes.LoadTestResult
	loadTestStatus  string
	loadTestSpec    string
}

func GenerateCheckOptions(req messages.TestnetWorkflowRequest, name, status, title, summary, text string, conclusion *string) github.CheckRunOptions {
	return github.CheckRunOptions{
		InstallationID: req.InstallationID,
		Owner:          req.Owner,
		Repo:           req.Repo,
		SHA:            req.SHA,
		Name:           name,
		Status:         util.StringPtr(status),
		Title:          util.StringPtr(title),
		Summary:        util.StringPtr(summary),
		Text:           text,
		Conclusion:     conclusion,
	}
}

func NewReport(ctx workflow.Context, name, title, summary string, req messages.TestnetWorkflowRequest) (*Report, error) {
	report := &Report{
		workflowRequest: req,
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
	var resp messages.CreateGitHubCheckResponse

	if err := workflow.ExecuteActivity(ctx, githubActivities.CreateGitHubCheck, messages.CreateGitHubCheckRequest{
		InstallationID: r.workflowRequest.InstallationID,
		Owner:          r.workflowRequest.Owner,
		Repo:           r.workflowRequest.Repo,
		SHA:            r.workflowRequest.SHA,
		Name:           r.name,
		Status:         &r.status,
		Title:          &r.title,
		Summary:        &r.summary,
		Conclusion:     nil,
	}).Get(ctx, &resp); err != nil {
		return -1, err
	}

	return int64(resp), nil
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

	return workflow.ExecuteActivity(ctx, githubActivities.UpdateGitHubCheck, messages.UpdateGitHubCheckRequest{
		CheckID:        r.checkId,
		InstallationID: r.workflowRequest.InstallationID,
		Owner:          r.workflowRequest.Owner,
		Repo:           r.workflowRequest.Repo,
		Name:           r.name,
		Status:         &r.status,
		Title:          &r.title,
		Summary:        &r.summary,
		Text:           output,
		Conclusion:     conclusion,
	}).Get(ctx, nil)
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

func (r *Report) SetBuildResult(ctx workflow.Context, buildResult messages.BuildDockerImageResponse) error {
	r.buildResult = buildResult
	return r.UpdateCheck(ctx)
}

func (r *Report) SetNodes(ctx workflow.Context, nodes []testnettypes.Node) error {
	r.nodes = nodes
	return r.UpdateCheck(ctx)
}

func (r *Report) UpdateLoadTest(ctx workflow.Context, status string, config string, results *catalysttypes.LoadTestResult) error {
	r.loadTestStatus = status
	r.loadTestSpec = config
	r.loadTestResults = results
	return r.UpdateCheck(ctx)
}

func (r *Report) SetDashboards(ctx workflow.Context, grafanaConfig types.GrafanaConfig, chainId string) error {
	urls := make(map[string]string)

	for _, dashboard := range grafanaConfig.Dashboards {
		url := fmt.Sprintf("%s/d/%s/%s?orgId=1&var-chain_id=%s&from=%d&to=%s&refresh=auto", grafanaConfig.URL, dashboard.ID, dashboard.Name, chainId, r.start.UnixMilli(), "now")
		urls[dashboard.HumanName] = url
	}

	r.dashboards = urls

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
		if r.loadTestSpec != "" {
			md.H2("Configuration")
			md.PlainText(r.loadTestSpec)
		}
	}

	// If we have results and no error, show them
	if r.loadTestResults != nil && r.loadTestResults.Error == "" {
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
		}

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

func (r *Report) addBuildResultToMarkdown(md *markdown.Markdown) {
	md.HorizontalRule()

	// Construct the entire collapsible section as raw HTML to avoid mixing markdown generation methods
	htmlContent := fmt.Sprintf(`
<details>
<summary><h2>Build Results</h2></summary>

<h3>Image tag: %s</h3>
<h3>Build logs:</h3>
<pre><code>%s</code></pre>
</details>
`, r.buildResult.FQDNTag, html.EscapeString(string(r.buildResult.Logs)))

	md.PlainText(htmlContent)
}

func (r *Report) addDashboardsToMarkdown(md *markdown.Markdown) {
	md.LF()
	md.HorizontalRule()
	md.LF()
	md.H1("Dashboards")
	md.LF()

	markdownLinks := make([]string, 0, len(r.dashboards))

	for name, url := range r.dashboards {
		markdownLinks = append(markdownLinks, markdown.Link(name, url))
	}

	md.BulletList(markdownLinks...)
	md.LF()
}

func (r *Report) Markdown() (string, error) {
	var buf bytes.Buffer

	md := markdown.NewMarkdown(&buf)

	if len(r.nodes) > 0 {
		r.addNodesToMarkdown(md)
	}

	if r.buildResult.FQDNTag != "" {
		r.addBuildResultToMarkdown(md)
	}

	if len(r.dashboards) != 0 {
		r.addDashboardsToMarkdown(md)
	}

	if r.loadTestStatus != "" || r.loadTestResults != nil {
		r.addLoadTestResultsToMarkdown(md)
	}

	if err := md.Build(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
