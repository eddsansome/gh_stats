package main

import "time"

type Page struct {
	Data struct {
		Repository struct {
			PullRequests struct {
				PageInfo struct {
					StartCursor string `json:"startCursor"`
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
				Edges []struct {
					Node struct {
						Title     string    `json:"title"`
						URL       string    `json:"url"`
						MergedAt  time.Time `json:"mergedAt"`
						CreatedAt time.Time `json:"createdAt"`
						Number    int       `json:"number"`
						Reviews   struct {
							Edges []struct {
								Node struct {
									State  string `json:"state"`
									Author struct {
										Login string `json:"login"`
									} `json:"author"`
								} `json:"node"`
							} `json:"edges"`
						} `json:"reviews"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"pullRequests"`
		} `json:"repository"`
	} `json:"data"`
}
