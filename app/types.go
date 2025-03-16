package app

import (
	"encoding/json"
	"fmt"
	github2 "github.com/google/go-github/v66/github"
	"strings"
)

type ValidatedPullRequest struct {
	InstallationID int64
	DeliveryID     string
	Number         int
	HeadSHA        string
	Owner          string
	Repo           string
	Action         string
}

func validatePullRequest(eventType, deliveryID string, payload []byte) (*ValidatedPullRequest, error) {
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

	return &ValidatedPullRequest{
		InstallationID: installationID,
		DeliveryID:     deliveryID,
		Number:         *pullRequest.Number,
		Owner:          owner.GetLogin(),
		Repo:           repo.GetName(),
		HeadSHA:        sha,
		Action:         event.GetAction(),
	}, nil
}

type ValidatedComment struct {
	InstallationID int64
	DeliveryID     string
	Owner          string
	Repo           string
	Body           string
	Sender         string
	IssueNumber    int

	IsOnPullRequest bool

	Action string
}

func validateComment(eventType, deliveryID string, payload []byte) (*ValidatedComment, error) {
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

	return &ValidatedComment{
		InstallationID:  installationID,
		DeliveryID:      deliveryID,
		Owner:           owner.GetLogin(),
		Repo:            repo.GetName(),
		Action:          event.GetAction(),
		Sender:          sender.GetLogin(),
		Body:            strings.TrimSpace(comment.GetBody()),
		IsOnPullRequest: issue.IsPullRequest(),
		IssueNumber:     issueNumber,
	}, nil
}
