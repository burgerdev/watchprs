# watchprs

This program periodically polls PRs of a GitHub repository, filters them for base branch and involved files, and sends a message to a Microsoft Teams channel for every match.

The approach to deduplication is simplistic: on startup, it records the largest PR number.
Every PR with a smaller number is considered to be handled already.
Once PRs with larger numbers pass the filters, the maximum number is updated.

There is pretty much no error handling - if a notification fails, it won't be retried.

Usage (remember not to pass secrets on the commandline!):

```sh
env GH_TOKEN=$token TEAMS_WEBHOOK=$hook watchprs --owner burgerdev --repo watchprs --base-re "^(main|master)$" --files-re "^pkg/.*$"
```
