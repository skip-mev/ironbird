package app

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/go-github/v66/github"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	temporalclient "go.temporal.io/sdk/client"
	"regexp"
)

var SubcommandRegex = regexp.MustCompile(`/ironbird ([^\s]*).*`)
var StartRegex = regexp.MustCompile(`/ironbird start ([^\s]*) ([^\s]*)`)

type CommandFunc func(context.Context, *ValidatedComment, string) error
type Command struct {
	Usage       string
	Description string
	Func        CommandFunc
}

func (a *App) generateInitialComment() (string, error) {
	var detailsOut bytes.Buffer
	var mdOut bytes.Buffer

	detailsMd := markdown.NewMarkdown(&detailsOut)
	detailsMd = detailsMd.PlainText("To use Ironbird, you can use the following commands:")

	var commandEntries []string

	for _, command := range a.commands {
		commandEntries = append(commandEntries, fmt.Sprintf("%s - %s", command.Usage, command.Description))
	}

	detailsMd = detailsMd.BulletList(commandEntries...)
	if err := detailsMd.Build(); err != nil {
		return "", err
	}

	md := markdown.NewMarkdown(&mdOut)
	md = md.Details("Ironbird - launch a network", detailsOut.String())

	if err := md.Build(); err != nil {
		return "", err
	}

	return mdOut.String(), nil
}

func (a *App) generatedFailedCommandComment(command string, err error) (string, error) {
	var mdOut bytes.Buffer

	md := markdown.NewMarkdown(&mdOut)
	md = md.PlainText(fmt.Sprintf("Ironbird failed to run command `%s`:", command))
	md = md.CodeBlocks("", err.Error())

	if err := md.Build(); err != nil {
		return "", err
	}

	return mdOut.String(), nil
}

func (a *App) SendInitialComment(ctx context.Context, pr *ValidatedPullRequest) error {
	c, err := a.cc.NewInstallationClient(pr.InstallationID)

	if err != nil {
		return err
	}

	commentBody, err := a.generateInitialComment()

	if err != nil {
		return err
	}

	_, _, err = c.Issues.CreateComment(ctx, pr.Owner, pr.Repo, pr.Number, &github.IssueComment{
		Body: &commentBody,
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *App) handleFailedCommand(ctx context.Context, comment *ValidatedComment, command string, commandErr error) error {
	if commandErr == nil {
		return fmt.Errorf("failed command cannot have a nil commandErr")
	}

	client, err := a.cc.NewInstallationClient(comment.InstallationID)

	if err != nil {
		return err
	}

	failedCommandCommentBody, err := a.generatedFailedCommandComment(command, commandErr)

	if err != nil {
		return err
	}

	_, _, err = client.Issues.CreateComment(ctx, comment.Owner, comment.Repo, comment.IssueNumber, &github.IssueComment{
		Body: &failedCommandCommentBody,
	})

	return err
}

func (a *App) HandleCommand(ctx context.Context, comment *ValidatedComment, command string) error {
	subcommandName := SubcommandRegex.FindAllStringSubmatch(command, -1)

	if len(subcommandName) != 1 {
		return fmt.Errorf("invalid command %s", command)
	}

	subcommand, ok := a.commands[subcommandName[0][1]]

	if !ok {
		return a.handleFailedCommand(ctx, comment, command, fmt.Errorf("unknown command %s", subcommandName[0][0]))
	}

	err := subcommand.Func(ctx, comment, command)

	if err != nil {
		return a.handleFailedCommand(ctx, comment, command, err)
	}

	return nil
}

func (a *App) commandStart(ctx context.Context, comment *ValidatedComment, command string) error {
	if !comment.IsOnPullRequest {
		return fmt.Errorf("command can only be run on pull requests")
	}

	client, err := a.cc.NewInstallationClient(comment.InstallationID)
	if err != nil {
		return err
	}

	isMember, _, err := client.Organizations.IsMember(ctx, comment.Owner, comment.Sender)

	if err != nil {
		return err
	}

	if !isMember {
		return fmt.Errorf("user %s is not a member of the organization", comment.Sender)
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
