package handler

import (
	"context"
	"log"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
	"github.com/google/go-github/v62/github"
)

type Teams struct {
	URL    string
	Prefix string

	client *goteamsnotify.TeamsClient
}

func (t *Teams) HandlePR(ctx context.Context, pr *github.PullRequest) {

	mstClient := t.client
	if mstClient == nil {
		mstClient = goteamsnotify.NewTeamsClient()
	}

	// Setup message card.
	msgCard := messagecard.NewMessageCard()
	title := "_untitled PR_"
	if pr.Title != nil {
		title = *pr.Title
	}
	msgCard.Title = t.Prefix + title
	msgCard.Text = "no URL available"
	if pr.HTMLURL != nil {
		msgCard.Text = *pr.HTMLURL
	}

	// Send the message with default timeout/retry settings.
	if err := mstClient.SendWithContext(ctx, t.URL, msgCard); err != nil {
		log.Printf("Failed to send message to teams: %v", err)
	}
}
