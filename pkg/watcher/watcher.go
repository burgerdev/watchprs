package watcher

import (
	"context"
	"fmt"
	"time"

	"github.com/google/go-github/v62/github"
)

type T struct {
	prs   PullRequestsService
	owner string
	repo  string

	acme int
}

func New(prs PullRequestsService, owner, repo string) *T {
	return &T{prs: prs, owner: owner, repo: repo}
}

func (t *T) Run(ctx context.Context, ticker <-chan time.Time, matcher Matcher, handlers []Handler) error {
	// Fetch once to set the high watermark.
	_, err := t.fetchPRs(ctx, MatcherFunc(func(ctx context.Context, pr *github.PullRequest) bool { return true }))
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker:
			prs, err := t.fetchPRs(ctx, matcher)
			if err != nil {
				// We fetched successfully once, so we might be successful again in the future - keep going.
				continue
			}
			for _, pr := range prs {
				for _, handler := range handlers {
					go handler.HandlePR(ctx, pr)
				}
			}
		}
	}
}

func (t *T) fetchPRs(ctx context.Context, matcher Matcher) ([]*github.PullRequest, error) {
	prs, resp, err := t.prs.List(ctx, t.owner, t.repo, &github.PullRequestListOptions{
		ListOptions: github.ListOptions{
			PerPage: 25,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("listing pull requests: %w (%v)", err, resp)
	}

	// Any PR in this batch that has a number higher than acme is interesting.
	acme := t.acme

	var out []*github.PullRequest
	for _, pr := range prs {
		if pr.Number == nil {
			// PRs ought to have a number, not dealing with this madness.
			continue
		}
		if !matcher.MatchPR(ctx, pr) {
			continue
		}
		if *pr.Number <= t.acme {
			continue
		}
		acme = max(acme, *pr.Number)
		out = append(out, pr)
	}
	t.acme = acme
	return out, nil
}
