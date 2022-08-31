package main

import "time"

type PullRequest struct {
	Id        int
	Reviewers map[string]bool
	MergedAt  time.Time
	CreatedAt time.Time
}
