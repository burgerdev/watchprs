package watcher

import (
	"context"

	"github.com/google/go-github/v62/github"
)

type PullRequestsService interface {
	List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
}

type Handler interface {
	HandlePR(context.Context, *github.PullRequest)
}

type HandlerFunc func(context.Context, *github.PullRequest)

func (f HandlerFunc) HandlePR(ctx context.Context, pr *github.PullRequest) {
	f(ctx, pr)
}

type Matcher interface {
	MatchPR(context.Context, *github.PullRequest) bool
}

type MatcherFunc func(context.Context, *github.PullRequest) bool

func (f MatcherFunc) MatchPR(ctx context.Context, pr *github.PullRequest) bool {
	return f(ctx, pr)
}
