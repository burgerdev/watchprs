package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/adaptivecard"
	"github.com/google/go-github/v62/github"
)

type teamsReceiver struct {
	Called bool
	Card   adaptivecard.Message
	Err    error
}

func (t *teamsReceiver) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	t.Called = true
	if err := json.NewDecoder(req.Body).Decode(&t.Card); err != nil {
		t.Err = err
	}
	rw.Write([]byte("1"))
}

func (t *teamsReceiver) Reset() {
	t.Called = false
	t.Card = adaptivecard.Message{}
	t.Err = nil
}

func TestTeams(t *testing.T) {

	receiver := &teamsReceiver{}
	server := httptest.NewServer(receiver)
	prefix := "foo: "

	teams := &Teams{
		URL:    server.URL,
		Prefix: prefix,
		client: goteamsnotify.NewTeamsClient(),
	}
	teams.client.SkipWebhookURLValidationOnSend(true)

	t.Run("standard request", func(t *testing.T) {
		pr := &github.PullRequest{
			Number:  ptr(42),
			Title:   ptr("fixing stuff"),
			HTMLURL: ptr("https://github.com/example/example/pull/42"),
		}

		teams.HandlePR(context.Background(), pr)

		if !receiver.Called {
			t.Fatalf("expected a call to the webhook URL")
		}
		if receiver.Err != nil {
			t.Fatalf("receiving webhook: %v", receiver.Err)
		}

		messages := receiver.Card.Attachments[0].Content.Body
		wantTitle := prefix + *pr.Title
		if messages[0].Text != wantTitle {
			t.Errorf("wrong title: got %q, want %q", messages[0].Text, wantTitle)
		}

		if !strings.Contains(messages[1].Text, *pr.HTMLURL) {
			t.Errorf("missing URL: expected %q in %q", *pr.HTMLURL, messages[1].Text)
		}
	})
}

func ptr[A any](a A) *A {
	return &a
}
