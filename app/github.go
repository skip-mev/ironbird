package app

import (
	"context"
	"fmt"
	"github.com/google/go-github/v66/github"
)

func (a *App) CreateComment(ctx context.Context, issue Issue, body string) (int64, error) {
	client, err := a.cc.NewInstallationClient(issue.InstallationID)

	if err != nil {
		return 0, err
	}

	comment, _, err := client.Issues.CreateComment(ctx, issue.Owner, issue.Repo, issue.Number, &github.IssueComment{
		Body: &body,
	})

	if err != nil {
		return 0, err
	}

	if comment == nil {
		return 0, fmt.Errorf("created issue is nil")
	}

	if comment.ID == nil {
		return 0, fmt.Errorf("created issue id is nil")
	}

	return *comment.ID, err
}
