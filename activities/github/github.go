package github

import (
	"context"
	"github.com/google/go-github/v66/github"
	"github.com/palantir/go-githubapp/githubapp"
	"github.com/skip-mev/ironbird/messages"
)

type NotifierActivity struct {
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
	Text           string
}

func (s *NotifierActivity) CreateGitHubCheck(ctx context.Context, req messages.CreateGitHubCheckRequest) (messages.CreateGitHubCheckResponse, error) {
	client, err := s.GithubClient.NewInstallationClient(req.InstallationID)
	if err != nil {
		return -1, err
	}

	var output *github.CheckRunOutput

	if req.Title != nil || req.Summary != nil {
		output = &github.CheckRunOutput{
			Title:   req.Title,
			Summary: req.Summary,
		}
	}

	checkRun, _, err := client.Checks.CreateCheckRun(ctx, req.Owner, req.Repo, github.CreateCheckRunOptions{
		Name:       req.Name,
		HeadSHA:    req.SHA,
		Status:     req.Status,
		Conclusion: req.Conclusion,
		Output:     output,
	})

	if err != nil {
		return -1, err
	}

	return messages.CreateGitHubCheckResponse(checkRun.GetID()), nil
}

func (s *NotifierActivity) UpdateGitHubCheck(ctx context.Context, req messages.UpdateGitHubCheckRequest) (messages.UpdateGitHubCheckResponse, error) {
	client, err := s.GithubClient.NewInstallationClient(req.InstallationID)
	if err != nil {
		return -1, err
	}

	var output *github.CheckRunOutput

	if req.Title != nil && req.Summary != nil {
		output = &github.CheckRunOutput{
			Title:   req.Title,
			Summary: req.Summary,
			Text:    &req.Text,
		}
	}

	checkRun, _, err := client.Checks.UpdateCheckRun(ctx, req.Owner, req.Repo, req.CheckID, github.UpdateCheckRunOptions{
		Name:       req.Name,
		Status:     req.Status,
		Conclusion: req.Conclusion,
		Output:     output,
	})

	if err != nil {
		return -1, err
	}

	return messages.UpdateGitHubCheckResponse(checkRun.GetID()), nil
}
