package app

import (
	"bytes"
	"context"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-github/v66/github"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	temporalclient "go.temporal.io/sdk/client"
	"regexp"
)

var SubcommandRegex = regexp.MustCompile(`/ironbird ([^\s]*).*`)
var StartRegex = regexp.MustCompile(`/ironbird start ([^\s]*) ([^\s]*)`)

var InitialCommentTemplate = fmt.Sprintf(`
<details>
<summary>Ironbird - launch a network</summary>

To launch a network, you can use the following commands:
- %s - to launch a testnet
- %s - to see available chains
- %s - to see available loadtests
</details>`,
	"`/ironbird start <chain> <loadtest>`",
	"`/ironbird chains`",
	"`/ironbird loadtests`")

func (a *App) SendInitialComment(ctx context.Context, pr *ValidatedPullRequest) error {
	c, err := a.cc.NewInstallationClient(pr.InstallationID)

	if err != nil {
		return err
	}

	_, resp, err := c.Issues.CreateComment(ctx, pr.Owner, pr.Repo, pr.Number, &github.IssueComment{
		Body: util.StringPtr(InitialCommentTemplate),
	})

	spew.Dump(resp)

	if err != nil {
		return err
	}

	return nil
}

func (a *App) HandleCommand(ctx context.Context, comment *ValidatedComment, command string) error {
	subcommand := SubcommandRegex.FindAllStringSubmatch(command, -1)

	if len(subcommand) != 1 {
		return fmt.Errorf("invalid command %s", command)
	}

	subcommandFunc, ok := a.commands[subcommand[0][1]]

	if !ok {
		return fmt.Errorf("unknown command %s", subcommand[0])
	}

	return subcommandFunc(ctx, comment, command)
}

func (a *App) commandStart(ctx context.Context, comment *ValidatedComment, command string) error {
	if !comment.IsOnPullRequest {
		return fmt.Errorf("command can only be run on pull requests")
	}

	client, err := a.cc.NewInstallationClient(comment.InstallationID)
	if err != nil {
		return err
	}

	pr, _, err := client.PullRequests.Get(ctx, comment.Owner, comment.Repo, comment.IssueNumber)

	if err != nil {
		return err
	}

	if pr == nil {
		return fmt.Errorf("no pull request found")
	}

	if pr.GetHead() == nil {
		return fmt.Errorf("no head found")
	}

	if pr.GetHead().SHA == nil {
		return fmt.Errorf("no head sha found")
	}
	args := StartRegex.FindAllStringSubmatch(command, -1)

	if len(args) != 1 {
		return fmt.Errorf("invalid command %s", command)
	}

	chainName := args[0][1]
	loadTestName := args[0][2]

	chain, ok := a.cfg.Chains[chainName]

	if !ok {
		return fmt.Errorf("unknown chain %s", chainName)
	}

	loadTest, ok := a.cfg.LoadTests[loadTestName]

	if !ok {
		return fmt.Errorf("unknown load test %s", loadTestName)
	}

	id := fmt.Sprintf("%s/%s/%s/pr-%d", chain.Name, comment.Owner, comment.Repo, comment.IssueNumber)

	_, err = a.temporalClient.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:        id,
		TaskQueue: testnet.TaskQueue,
	}, testnet.Workflow, testnet.WorkflowOptions{
		InstallationID: comment.InstallationID,
		Owner:          comment.Owner,
		Repo:           comment.Repo,
		SHA:            *pr.Head.SHA,
		ChainConfig:    chain,
		LoadTestConfig: &loadTest,
	})

	if err != nil {
		fmt.Println("failed to execute workflow", err)
		return err
	}

	return nil
}

func (a *App) commandChains(ctx context.Context, comment *ValidatedComment, _ string) error {
	client, err := a.cc.NewInstallationClient(comment.InstallationID)

	if err != nil {
		return err
	}

	var mdOut bytes.Buffer

	md := markdown.NewMarkdown(&mdOut)

	var entries []string
	for _, chain := range a.cfg.Chains {
		entries = append(entries, fmt.Sprintf("`%s` (version `%s`)", chain.Name, chain.Version))
	}

	md = md.H3("Ironbird - available chains")
	md = md.BulletList(entries...)

	if err := md.Build(); err != nil {
		return err
	}

	_, _, err = client.Issues.CreateComment(ctx, comment.Owner, comment.Repo, comment.IssueNumber, &github.IssueComment{
		Body: util.StringPtr(mdOut.String()),
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *App) commandLoadTests(ctx context.Context, comment *ValidatedComment, command string) error {
	client, err := a.cc.NewInstallationClient(comment.InstallationID)

	if err != nil {
		return err
	}

	var mdOut bytes.Buffer

	md := markdown.NewMarkdown(&mdOut)

	var entries []string
	for name, loadTest := range a.cfg.LoadTests {
		entries = append(entries, fmt.Sprintf("`%s` - `%s`", name, loadTest.Description))
	}

	md = md.H3("Ironbird - available loadtests")
	md = md.BulletList(entries...)

	if err := md.Build(); err != nil {
		return err
	}

	_, _, err = client.Issues.CreateComment(ctx, comment.Owner, comment.Repo, comment.IssueNumber, &github.IssueComment{
		Body: util.StringPtr(mdOut.String()),
	})

	return nil
}
