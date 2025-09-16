package handler

import (
	"context"
	"fmt"
	"log"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/adaptivecard"
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
		// Workaround for https://github.com/atc0005/go-teams-notify/issues/310.
		validPattern := `^https:\/\/(?:.*)(:?\.azure-api|logic\.azure|api\.powerplatform)\.(?:com|net)`
		mstClient = goteamsnotify.NewTeamsClient().AddWebhookURLValidationPatterns(validPattern)
	}
	title := "_untitled PR_"
	if pr.Title != nil {
		title = *pr.Title
	}
	text := "no URL available"
	if pr.HTMLURL != nil {
		text = fmt.Sprintf("[%s](%s)", title, *pr.HTMLURL)
	}

	msg, err := adaptivecard.NewSimpleMessage(text, t.Prefix+title, false)
	if err != nil {
		log.Printf("Failed to create simple message: %v", err)
		return
	}

	// Send the message with default timeout/retry settings.
	if err := mstClient.SendWithContext(ctx, t.URL, msg); err != nil {
		log.Printf("Failed to send message to teams: %v", err)
	}
}
