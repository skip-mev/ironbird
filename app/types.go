package app

import (
	"encoding/json"
	"fmt"
	github2 "github.com/google/go-github/v66/github"
	"strings"
)

type PullRequest struct {
	Issue
	DeliveryID string
	HeadSHA    string
	Action     string
}

func parsePullRequestEvent(eventType, deliveryID string, payload []byte) (*PullRequest, error) {
	if eventType != "pull_request" {
		return nil, fmt.Errorf("event type %s is not a pull request", eventType)
	}

	var event github2.PullRequestEvent

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	installation := event.GetInstallation()

	if installation == nil {
		return nil, fmt.Errorf("installation id for %s is nil", deliveryID)
	}

	installationID := installation.GetID()

	repo := event.GetRepo()

	if repo == nil {
		return nil, fmt.Errorf("repo for %s is nil", deliveryID)
	}

	owner := repo.GetOwner()

	if owner == nil {
		return nil, fmt.Errorf("owner for %s is nil", deliveryID)
	}

	pullRequest := event.GetPullRequest()

	if pullRequest == nil {
		return nil, fmt.Errorf("pull request for %s is nil", deliveryID)
	}

	if pullRequest.Number == nil {
		return nil, fmt.Errorf("pull request id for %s is nil", deliveryID)
	}

	head := pullRequest.GetHead()

	if head == nil {
		return nil, fmt.Errorf("head for %s is nil", deliveryID)
	}

	sha := head.GetSHA()

	return &PullRequest{
		DeliveryID: deliveryID,
		Issue: Issue{
			InstallationID: installationID,
			IsPullRequest:  true,
			Number:         *pullRequest.Number,
			Owner:          owner.GetLogin(),
			Repo:           repo.GetName(),
		},
		HeadSHA: sha,
		Action:  event.GetAction(),
	}, nil
}

type Issue struct {
	InstallationID int64

	Owner         string
	Repo          string
	Number        int
	IsPullRequest bool
}

type Comment struct {
	Issue
	DeliveryID string
	Sender     string
	Body       string

	Action string
}

func parseComment(eventType, deliveryID string, payload []byte) (*Comment, error) {
	if eventType != "issue_comment" {
		return nil, fmt.Errorf("event type %s is not a comment", eventType)
	}

	var event github2.IssueCommentEvent

	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, err
	}

	installation := event.GetInstallation()

	if installation == nil {
		return nil, fmt.Errorf("installation id for %s is nil", deliveryID)
	}

	installationID := installation.GetID()

	repo := event.GetRepo()

	if repo == nil {
		return nil, fmt.Errorf("repo for %s is nil", deliveryID)
	}

	owner := repo.GetOwner()

	if owner == nil {
		return nil, fmt.Errorf("owner for %s is nil", deliveryID)
	}

	comment := event.GetComment()
	if comment == nil {
		return nil, fmt.Errorf("comment for %s is nil", deliveryID)
	}

	issue := event.GetIssue()
	if issue == nil {
		return nil, fmt.Errorf("issue for %s is nil", deliveryID)
	}

	issueNumber := issue.GetNumber()

	sender := event.GetSender()

	if sender == nil {
		return nil, fmt.Errorf("sender for %s is nil", deliveryID)
	}

	return &Comment{
		DeliveryID: deliveryID,
		Issue: Issue{
			InstallationID: installationID,
			Number:         issueNumber,
			Owner:          owner.GetLogin(),
			Repo:           repo.GetName(),
			IsPullRequest:  issue.IsPullRequest(),
		},
		Action: event.GetAction(),
		Sender: sender.GetLogin(),
		Body:   strings.TrimSpace(comment.GetBody()),
	}, nil
}
