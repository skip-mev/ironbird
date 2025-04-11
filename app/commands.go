package app

import (
	"bytes"
	"context"
	"fmt"
	"regexp"

	"github.com/google/go-github/v66/github"
	"github.com/nao1215/markdown"
	"github.com/skip-mev/ironbird/types"
	testnettypes "github.com/skip-mev/ironbird/types/testnet"
	"github.com/skip-mev/ironbird/util"
	"github.com/skip-mev/ironbird/workflows/testnet"
	temporalclient "go.temporal.io/sdk/client"
	"gopkg.in/yaml.v3"
)

var SubcommandRegex = regexp.MustCompile(`/ironbird ([^\s]*).*`)
var StartRegex = regexp.MustCompile(`/ironbird start ([^\s]*) ?([^\s]*)`)
var CustomConfigRegex = regexp.MustCompile(`--load-test-config=(\{.*\})`)

type CommandFunc func(context.Context, *Comment, string) error
type Command struct {
	Usage       string
	Description string
	Func        CommandFunc
}

func (a *App) generateInitialComment() (string, error) {
	var detailsOut bytes.Buffer
	var mdOut bytes.Buffer
	var customConfigOut bytes.Buffer

	detailsMd := markdown.NewMarkdown(&detailsOut)
	detailsMd = detailsMd.PlainText("To use Ironbird, you can use the following commands:").LF()

	var commandEntries []string

	for _, command := range a.commands {
		commandEntries = append(commandEntries, fmt.Sprintf("%s - %s", command.Usage, command.Description))
	}

	detailsMd = detailsMd.BulletList(commandEntries...)

	customConfigMd := markdown.NewMarkdown(&customConfigOut)
	customConfigMd = customConfigMd.PlainText("**Custom Load Test Configurations**").LF()
	customConfigMd = customConfigMd.PlainText("You can provide a custom load test configuration using the `--load-test-config=` flag:").LF()
	customConfigMd = customConfigMd.CodeBlocks("", `/ironbird start cosmos --load-test-config={
  "block_gas_limit_target": 0.75,
  "num_of_blocks": 50,
  "msgs": [
    {"weight": 0.3, "type": "MsgSend"},
    {"weight": 0.3, "type": "MsgMultiSend"},
	{"weight": 0.4, "type": "MsgArr", "ContainedType": "MsgSend", "NumMsgs": 3300}
  ]
}`)
	customConfigMd = customConfigMd.PlainText("Use `/ironbird loadtests` to see more examples.").LF()

	if err := customConfigMd.Build(); err != nil {
		return "", err
	}

	if err := detailsMd.Build(); err != nil {
		return "", err
	}

	md := markdown.NewMarkdown(&mdOut)
	md = md.Details("Ironbird - launch a network", detailsOut.String())
	md = md.Details("Custom Load Test Configuration", customConfigOut.String())

	if err := md.Build(); err != nil {
		return "", err
	}

	return mdOut.String(), nil
}

func (a *App) generatedFailedCommandComment(command string, err error) (string, error) {
	var mdOut bytes.Buffer

	md := markdown.NewMarkdown(&mdOut)
	md = md.PlainText(fmt.Sprintf("Ironbird failed to run command `%s`:", command)).LF()
	md = md.CodeBlocks("", err.Error())

	if err := md.Build(); err != nil {
		return "", err
	}

	return mdOut.String(), nil
}

func (a *App) generateStartedTestComment(chainConfig types.ChainsConfig, loadTestConfig types.LoadTestConfig, workflowId, runId string, runnerType testnettypes.RunnerType) (string, error) {
	var mdOut bytes.Buffer
	var chainDetails bytes.Buffer
	var loadTestDetails bytes.Buffer

	chainMd := markdown.NewMarkdown(&chainDetails)
	chainMd = chainMd.LF().PlainText(fmt.Sprintf("Chain: `%s`", chainConfig.Name)).LF()
	chainMd = chainMd.PlainText(fmt.Sprintf("Version: `%s`", chainConfig.Version)).LF()
	chainMd = chainMd.PlainText(fmt.Sprintf("Runner: `%s`", runnerType)).LF()
	chainMd = chainMd.PlainText(fmt.Sprintf("Workflow ID: `%s`", workflowId)).LF()
	chainMd = chainMd.PlainText(fmt.Sprintf("Run ID: `%s`", runId)).LF()

	if err := chainMd.Build(); err != nil {
		return "", err
	}

	loadTestMd := markdown.NewMarkdown(&loadTestDetails)
	loadTestMd = loadTestMd.LF().PlainText(fmt.Sprintf("Load test: `%s`", loadTestConfig.Name)).LF()
	loadTestMd = loadTestMd.PlainText(fmt.Sprintf("Description: `%s`", loadTestConfig.Description)).LF()

	if err := loadTestMd.Build(); err != nil {
		return "", err
	}

	md := markdown.NewMarkdown(&mdOut)
	md = md.PlainText(fmt.Sprintf("Ironbird has started a testnet for chain `%s` using loadtest `%s` with runner `%s`", chainConfig.Name, loadTestConfig.Name, runnerType)).LF()
	md = md.Details("Chain details", chainDetails.String())
	md = md.Details("Load test details", loadTestDetails.String())

	if err := md.Build(); err != nil {
		return "", err
	}

	return mdOut.String(), nil
}

func (a *App) SendInitialComment(ctx context.Context, pr *PullRequest) error {
	commentBody, err := a.generateInitialComment()

	if err != nil {
		return err
	}

	if _, err := a.CreateComment(ctx, pr.Issue, commentBody); err != nil {
		return err
	}

	return nil
}

func (a *App) handleFailedCommand(ctx context.Context, comment *Comment, command string, commandErr error) error {
	if commandErr == nil {
		return fmt.Errorf("failed command cannot have a nil commandErr")
	}

	failedCommandCommentBody, err := a.generatedFailedCommandComment(command, commandErr)

	if err != nil {
		return err
	}

	if _, err := a.CreateComment(ctx, comment.Issue, failedCommandCommentBody); err != nil {
		return err
	}

	return err
}

func (a *App) HandleCommand(ctx context.Context, comment *Comment, command string) error {
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

func (a *App) parseCustomLoadTestConfig(yamlStr string) (*types.LoadTestConfig, error) {
	var customConfig types.LoadTestConfig
	if err := yaml.Unmarshal([]byte(yamlStr), &customConfig); err != nil {
		return nil, fmt.Errorf("failed to parse custom load test config: %w", err)
	}

	if customConfig.Name == "" {
		customConfig.Name = "custom"
	}

	if customConfig.BlockGasLimitTarget <= 0 || customConfig.BlockGasLimitTarget > 1 {
		return nil, fmt.Errorf("block_gas_limit_target must be between 0 and 1")
	}

	if customConfig.NumOfBlocks <= 0 {
		return nil, fmt.Errorf("num_of_blocks must be greater than 0")
	}

	if len(customConfig.Msgs) == 0 {
		return nil, fmt.Errorf("at least one message type must be specified")
	}

	totalWeight := 0.0
	for _, msg := range customConfig.Msgs {
		totalWeight += msg.Weight
	}

	if totalWeight != 1.0 {
		return nil, fmt.Errorf("message weights must sum to exactly 1.0 (got %.2f)", totalWeight)
	}

	return &customConfig, nil
}

func (a *App) commandStart(ctx context.Context, comment *Comment, command string) error {
	if !comment.Issue.IsPullRequest {
		return fmt.Errorf("command can only be run on pull requests")
	}

	client, err := a.cc.NewInstallationClient(comment.InstallationID)
	if err != nil {
		return err
	}

	isMember, _, err := client.Organizations.IsMember(ctx, comment.Issue.Owner, comment.Sender)

	if err != nil {
		return err
	}

	if !isMember {
		return fmt.Errorf("user %s is not a member of the organization", comment.Sender)
	}

	pr, _, err := client.PullRequests.Get(ctx, comment.Issue.Owner, comment.Issue.Repo, comment.Issue.Number)

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

	customConfig := CustomConfigRegex.FindStringSubmatch(command)
	hasCustomConfig := len(customConfig) > 1

	// If custom config is provided, loadTestName should be empty
	if hasCustomConfig && loadTestName != "" {
		return fmt.Errorf("when using --load-test-config, do not specify a load test mode (e.g., 'full')")
	}

	// If no custom config is provided, loadTestName is required
	if !hasCustomConfig && loadTestName == "" {
		return fmt.Errorf("load test mode is required when not using --load-test-config")
	}

	// Set runner type to DigitalOcean by default when using Github CLI
	runnerType := testnettypes.DigitalOcean

	chain, ok := a.cfg.Chains[chainName]

	if !ok {
		return fmt.Errorf("unknown chain %s", chainName)
	}

	var loadTest types.LoadTestConfig
	if hasCustomConfig {
		customLoadTest, err := a.parseCustomLoadTestConfig(customConfig[1])
		if err != nil {
			return err
		}
		loadTest = *customLoadTest
	} else {
		configuredLoadTest, ok := a.cfg.LoadTests[loadTestName]
		if !ok {
			return fmt.Errorf("unknown load test %s", loadTestName)
		}
		loadTest = configuredLoadTest
	}

	id := fmt.Sprintf("%s/%s/%s/pr-%d", chain.Name, comment.Issue.Owner, comment.Issue.Repo, comment.Issue.Number)

	workflow, err := a.temporalClient.ExecuteWorkflow(ctx, temporalclient.StartWorkflowOptions{
		ID:        id,
		TaskQueue: testnet.TaskQueue,
	}, testnet.Workflow, testnet.WorkflowOptions{
		InstallationID: comment.InstallationID,
		Owner:          comment.Issue.Owner,
		Repo:           comment.Issue.Repo,
		SHA:            *pr.Head.SHA,
		ChainConfig:    chain,
		LoadTestConfig: &loadTest,
		RunnerType:     runnerType,
	})

	if err != nil {
		fmt.Println("failed to execute workflow", err)
		return err
	}

	commentBody, err := a.generateStartedTestComment(chain, loadTest, workflow.GetID(), workflow.GetRunID(), runnerType)

	if err != nil {
		return err
	}

	if _, err := a.CreateComment(ctx, comment.Issue, commentBody); err != nil {
		return err
	}

	return nil
}

func (a *App) commandChains(ctx context.Context, comment *Comment, _ string) error {
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

	_, _, err = client.Issues.CreateComment(ctx, comment.Issue.Owner, comment.Issue.Repo, comment.Issue.Number, &github.IssueComment{
		Body: util.StringPtr(mdOut.String()),
	})

	if err != nil {
		return err
	}

	return nil
}

func (a *App) commandLoadTests(ctx context.Context, comment *Comment, _ string) error {
	var mdOut bytes.Buffer
	var exampleOut bytes.Buffer

	md := markdown.NewMarkdown(&mdOut)
	exampleMd := markdown.NewMarkdown(&exampleOut)

	var entries []string
	for name, loadTest := range a.cfg.LoadTests {
		entries = append(entries, fmt.Sprintf("`%s` - `%s`", name, loadTest.Description))
	}

	entries = append(entries, "`custom` - Use the `--load-test-config={...}` flag with the start command to provide an inline YAML configuration")

	md = md.H3("Ironbird - available loadtests")
	md = md.BulletList(entries...)

	exampleMd = exampleMd.PlainText("Example of using a custom load test configuration:").LF()
	exampleMd = exampleMd.CodeBlocks("", `/ironbird start cosmos --load-test-config={
  "name": "custom-test",
  "description": "Custom load test with 1000 transactions per block",
  "num_of_txs": 1000,
  "num_of_blocks": 50,
  "msgs": [
    {"weight": 0.3, "type": "MsgSend"},
    {"weight": 0.3, "type": "MsgMultiSend"},
	{"weight": 0.4, "type": "MsgArr", "ContainedType": "MsgSend", "NumMsgs": 50 }
  ]
}`)

	if err := exampleMd.Build(); err != nil {
		return err
	}

	if err := md.Build(); err != nil {
		return err
	}

	finalOutput := mdOut.String() + "\n\n" + exampleOut.String()

	if _, err := a.CreateComment(ctx, comment.Issue, finalOutput); err != nil {
		return err
	}

	return nil
}
