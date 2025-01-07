package pipeline

import (
	"context"
	"github.com/google/go-github/v66/github"
	"github.com/palantir/go-githubapp/githubapp"
)

type GithubNotifierActivity struct {
	GithubClient githubapp.ClientCreator
}

type CheckRunOptions struct {
	InstallationID int64
	Owner          string
	Repo           string
	Name           string
	SHA            string
	Status         *string
	Conclusion     *string
	Title          *string
	Summary        *string
}

func (s *GithubNotifierActivity) CreateCheck(ctx context.Context, opts CheckRunOptions) (int64, error) {
	client, err := s.GithubClient.NewInstallationClient(opts.InstallationID)
	if err != nil {
		return -1, err
	}

	var output *github.CheckRunOutput

	if opts.Title != nil || opts.Summary != nil {
		output = &github.CheckRunOutput{
			Title:   opts.Title,
			Summary: opts.Summary,
		}
	}

	checkRun, _, err := client.Checks.CreateCheckRun(ctx, opts.Owner, opts.Repo, github.CreateCheckRunOptions{
		Name:       opts.Name,
		HeadSHA:    opts.SHA,
		Status:     opts.Status,
		Conclusion: opts.Conclusion,
		Output:     output,
	})

	if err != nil {
		return -1, err
	}

	return checkRun.GetID(), nil
}

func (s *GithubNotifierActivity) UpdateCheck(ctx context.Context, id int64, opts CheckRunOptions) (int64, error) {
	client, err := s.GithubClient.NewInstallationClient(opts.InstallationID)
	if err != nil {
		return -1, err
	}

	var output *github.CheckRunOutput

	if opts.Title != nil && opts.Summary != nil {
		output = &github.CheckRunOutput{
			Title:   opts.Title,
			Summary: opts.Summary,
		}
	}

	checkRun, _, err := client.Checks.UpdateCheckRun(ctx, opts.Owner, opts.Repo, id, github.UpdateCheckRunOptions{
		Name:       opts.Name,
		Status:     opts.Status,
		Conclusion: opts.Conclusion,
		Output:     output,
	})

	if err != nil {
		return -1, err
	}

	return checkRun.GetID(), nil
}
