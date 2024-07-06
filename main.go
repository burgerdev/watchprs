package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"time"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
	"github.com/burgerdev/watchprs/pkg/handler"
	"github.com/burgerdev/watchprs/pkg/watcher"
	"github.com/google/go-github/v62/github"
)

const (
	tokenEnvVar    = "GH_TOKEN"
	teamsURLEnvVar = "TEAMS_WEBHOOK"
)

var (
	owner  = flag.String("owner", "", "Github repository owner")
	repo   = flag.String("repo", "", "Github repository")
	base   = flag.String("base-re", ".*", "regular expression for the PR's target branch")
	files  = flag.String("files-re", ".*", "regular expression for files of interest")
	period = flag.Duration("period", time.Minute, "time between GitHub API calls")
	debug  = flag.Bool("debug", false, "print debug information")
)

type config struct {
	gh       *github.Client
	matcher  watcher.Matcher
	handlers []watcher.Handler
}

func initialize() *config {
	flag.Parse()

	if *owner == "" || *repo == "" {
		log.Fatalf("--owner and --repo must be set")
	}

	token := os.Getenv(tokenEnvVar)
	if token == "" {
		log.Fatalf("Environment variable %q must be set!", tokenEnvVar)
	}
	filesRE, err := regexp.Compile(*files)
	if err != nil {
		log.Fatalf("Could not compile --files-re=%q: %v", *files, err)
	}
	baseRE, err := regexp.Compile(*base)
	if err != nil {
		log.Fatalf("Could not compile --base-re=%q: %v", *base, err)
	}
	matcher := watcher.MatcherFunc(func(_ context.Context, pr *github.PullRequest) bool {
		return matchPR(pr, baseRE, filesRE)
	})

	handlers := []watcher.Handler{}

	teamsURL := os.Getenv(teamsURLEnvVar)
	if teamsURL != "" {
		log.Println("Setting up Microsoft Teams integration")
		handlers = append(handlers, &handler.Teams{URL: teamsURL, Prefix: fmt.Sprintf("%s/%s: ", *owner, *repo)})
	}

	if *debug {
		handlers = append(handlers, watcher.HandlerFunc(func(ctx context.Context, pr *github.PullRequest) {
			url := "<no URL in PR>"
			if pr.HTMLURL != nil {
				url = *pr.HTMLURL
			}
			title := "<no title>"
			if pr.Title != nil {
				title = *pr.Title
			}
			log.Printf("Handling PR %s: %q", url, title)
		}))
	}

	return &config{
		gh:       github.NewClient(nil).WithAuthToken(token),
		matcher:  matcher,
		handlers: handlers,
	}
}

func main() {
	cfg := initialize()
	ctx, cancel := context.WithCancel(context.Background())

	t := time.NewTicker(*period)

	w := watcher.New(cfg.gh.PullRequests, *owner, *repo)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	go func() {
		if err := w.Run(ctx, t.C, cfg.matcher, cfg.handlers); err != nil {
			if err != context.Canceled {
				log.Fatal(err)
			}
			sigs <- os.Interrupt
		}
	}()

	<-sigs
	cancel()
}

func matchPR(pr *github.PullRequest, branchRE, filesRE *regexp.Regexp) bool {
	if pr.DiffURL == nil {
		return false
	}
	if pr.Base == nil || !branchRE.MatchString(pr.Base.GetLabel()) {
		return false
	}
	diffs, err := fetchDiff(*pr.DiffURL)
	if err != nil {
		log.Printf("error fetching diff for PR %d: %v", *pr.Number, err)
		return false
	}
	for _, diff := range diffs {
		if filesRE.MatchString(diff.NewName) {
			return true
		}
		if filesRE.MatchString(diff.OldName) {
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
