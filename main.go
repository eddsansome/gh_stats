package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"time"
)

var token = flag.String("t", "", "Pass in your GH access token pls!")

func main() {

	flag.Parse()

	if *token == "" {
		log.Fatal("NEED TOKEN!")
	}

	prs := []PullRequest{}

	// we start with an empty string as we page back from the last page
	allPRs := getPrs("", prs)

	filteredPrs := filterPRs(allPRs)

	// sort the array by date
	sort.Slice(filteredPrs, func(i, j int) bool {
		return filteredPrs[i].MergedAt.Before(filteredPrs[j].MergedAt)
	})

	fmt.Println("PR,reviewers,opened,merged,cycle time")
	// fmt.Println("Number of PR's closed in August: ", len(filteredPrs))
	for _, pr := range filteredPrs {

		reviewers := ""

		for reviewerName := range pr.Reviewers {

			reviewers += reviewerName + " | "

		}

		cycleTime := pr.MergedAt.Sub(pr.CreatedAt)

		fmt.Printf("%d,%d,%v,%v,%v\n", pr.Id, len(pr.Reviewers), pr.CreatedAt.Format("2006-01-02"), pr.MergedAt.Format("2006-01-02"), cycleTime)
	}

}

func filterPRs(prs []PullRequest) []PullRequest {
	filtered := []PullRequest{}

	// This will filter all PR's in August - Adjust accordingly
	for _, pr := range prs {
		if pr.MergedAt.Before(time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC)) || pr.MergedAt.After(time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)) {
			continue
		}

		filtered = append(filtered, pr)
	}
	return filtered
}

// recursively page through the Github GraphQL endpoint
func getPrs(before string, prs []PullRequest) []PullRequest {

	// rate limiting hehe
	time.Sleep(time.Second * 2)

	// if merged at is over 1 month ago, return the PR's
	// guard clause
	if len(prs) > 1 {
		lastPr := prs[len(prs)-1]

		if lastPr.MergedAt.Before(time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC)) {
			return prs
		}

	}

	if before == "" {
		before = "null"
	} else {
		before = fmt.Sprintf("\"%s\"", before)
	}

	// so, we need to get the first page of results of 100 PR's
	// then we need to use the startCursor, with the before, to paginate backwards
	jsonData := map[string]string{
		"query": fmt.Sprintf(`{ repository(owner: "smartpension", name: "api") {
		  pullRequests(last: 100, states: MERGED, orderBy: {field: UPDATED_AT, direction: ASC}, before: %s) {
			pageInfo {
			  startCursor
			  hasNextPage
			  endCursor
			}
			edges {
			  node {
				title
				url
				mergedAt
				createdAt
				number
				reviews(first: 100) {
				  edges {
					node {
					  state
					  author {
						login
					  }
					}
				  }
				}
			  }
			}
		  }
		}
	  }
	`, before)}

	jsonValue, err := json.Marshal(jsonData)

	if err != nil {
		panic(err)
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", bytes.NewBuffer(jsonValue))

	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))
	res, err := http.DefaultClient.Do(req)

	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	var page Page

	if err := json.Unmarshal(body, &page); err != nil {
		panic(err)
	}

	for _, pr := range page.Data.Repository.PullRequests.Edges {
		p := PullRequest{CreatedAt: pr.Node.CreatedAt, Id: pr.Node.Number, MergedAt: pr.Node.MergedAt, Reviewers: map[string]bool{}}
		for _, r := range pr.Node.Reviews.Edges {
			if r.Node.State == "APPROVED" {
				p.Reviewers[r.Node.Author.Login] = true
			}
		}
		prs = append(prs, p)
	}

	return getPrs(page.Data.Repository.PullRequests.PageInfo.StartCursor, prs)

}
