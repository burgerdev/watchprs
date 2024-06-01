package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"

	goteamsnotify "github.com/atc0005/go-teams-notify/v2"
	"github.com/atc0005/go-teams-notify/v2/messagecard"
	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/google/go-github/v62/github"
)

const (
	tokenEnvVar    = "GH_TOKEN"
	teamsURLEnvVar = "TEAMS_WEBHOOK"
)

var (
	owner   = flag.String("owner", "", "Github repository owner")
	repo    = flag.String("repo", "", "Github repository")
	base    = flag.String("base", "main", "PR target branch")
	filesRE = flag.String("files-re", ".*", "regular expression for files of interest")
)

type Config struct {
	Token    string
	TeamsURL string
	Repo     string
	Owner    string
	Base     string
	RE       *regexp.Regexp
}

func initialize() *Config {
	flag.Parse()
	token := os.Getenv(tokenEnvVar)
	if token == "" {
		log.Fatalf("Environment variable %q must be set!", tokenEnvVar)
	}
	teamsURL := os.Getenv(teamsURLEnvVar)
	if token == "" {
		log.Fatalf("Environment variable %q must be set!", teamsURLEnvVar)
	}

	if *owner == "" || *repo == "" {
		log.Fatalf("--owner and --repo must be set")
	}
	re, err := regexp.Compile(*filesRE)
	if err != nil {
		log.Fatalf("Could not compile --files-re=%q: %v", *filesRE, err)
	}

	return &Config{
		Token:    token,
		TeamsURL: teamsURL,
		Repo:     *repo,
		Owner:    *owner,
		Base:     *base,
		RE:       re,
	}
}

func main() {
	cfg := initialize()
	gh := github.NewClient(nil).WithAuthToken(cfg.Token)

	// Populate cache
	prs := make(map[int]*github.PullRequest)
	for _, pr := range fetchPRs(gh, cfg) {
		prs[*pr.Number] = pr
	}

	iter := func() {
		for _, pr := range fetchPRs(gh, cfg) {
			if _, ok := prs[*pr.Number]; ok {
				continue
			}
			handleNewPR(pr, cfg)
			prs[*pr.Number] = pr
		}
	}

	iter()
	tic := time.NewTicker(300 * time.Second)
	for range tic.C {
		iter()
	}
}

func handleNewPR(pr *github.PullRequest, cfg *Config) {
	log.Printf("===== NEW PR: %s =====", *pr.URL)

	// Initialize a new Microsoft Teams client.
	mstClient := goteamsnotify.NewTeamsClient()

	// Setup message card.
	msgCard := messagecard.NewMessageCard()
	msgCard.Title = fmt.Sprintf("%s/%s: %s", cfg.Owner, cfg.Repo, *pr.Title)
	msgCard.Text = *pr.HTMLURL
	// msgCard.ThemeColor = "#DF813D"

	// Send the message with default timeout/retry settings.
	if err := mstClient.Send(cfg.TeamsURL, msgCard); err != nil {
		log.Printf("failed to send message to teams: %v", err)
	}
}

func fetchPRs(gh *github.Client, cfg *Config) []*github.PullRequest {
	prs, resp, err := gh.PullRequests.List(context.Background(), *owner, *repo, &github.PullRequestListOptions{
		Base: *base,
		ListOptions: github.ListOptions{
			PerPage: 25,
		},
	})
	if err != nil {
		log.Printf("Listing pull requests failed: %v (%v)", err, resp)
	}
	log.Printf("Github response: %v", resp)

	var out []*github.PullRequest
	for _, pr := range prs {
		if !matchesFiles(pr, cfg.RE) {
			continue
		}
		out = append(out, pr)
	}
	return out
}

func matchesFiles(pr *github.PullRequest, re *regexp.Regexp) bool {
	if pr.DiffURL == nil {
		return false
	}
	diffs, err := fetchDiff(*pr.DiffURL)
	if err != nil {
		log.Printf("error fetching diff for PR %d: %v", *pr.Number, err)
		return false
	}
	for _, diff := range diffs {
		if re.MatchString(diff.NewName) {
			return true
		}
		if re.MatchString(diff.OldName) {
			return true
		}
	}
	return false
}

func fetchDiff(u string) ([]*gitdiff.File, error) {
	resp, err := http.DefaultClient.Get(u)
	if err != nil {
		return nil, fmt.Errorf("could not fetch %q: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching %q returned status %d", u, resp.StatusCode)
	}

	files, _, err := gitdiff.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not parse diff: %w", err)
	}
	return files, nil
}
