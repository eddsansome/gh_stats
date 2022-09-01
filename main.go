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

// Rather than rely on an ENV parser, we ask the user to pass in their token via the command line
// This could be changed to use Dotenv
var token = flag.String("t", "", "Pass in your GH access token pls!")

func main() {

	// Here we parse and assign the command line flags
	flag.Parse()

	// if the token is not supplied, we exit ASAP :)
	if *token == "" {
		log.Fatal("NEED TOKEN!")
	}

	// Create a slice (Ruby array) to hold our PR's
	prs := []PullRequest{}

	// getPrs takes a (string, slice of PRs) and RECURSIVELY calls the Github GraphQL endpoint
	// The string is the page that we want to start parsing the GraphQL from
	// We start with an empty string as we page back from the last page
	// The slice will be empty at this point too
	allPRs := getPrs("", prs)

	// Filter the PR's (in this case by date (August))
	filteredPrs := filterPRs(allPRs)

	// Sort the array in place by date
	sort.Slice(filteredPrs, func(i, j int) bool {
		return filteredPrs[i].MergedAt.Before(filteredPrs[j].MergedAt)
	})

	// Print out the first line of the CSV to stdout
	fmt.Println("PR,reviewers,opened,merged,cycle time")

	// Iterate over all of the PR's and print the selected data out to stdout
	for _, pr := range filteredPrs {

		// We can create a list of approvers here
		// Not using this in the CSV at the moment
		reviewers := ""
		for reviewerName := range pr.Reviewers {
			reviewers += reviewerName + " | "
		}

		// Calculate the cycle time
		cycleTime := pr.MergedAt.Sub(pr.CreatedAt)

		// Print out all of the information to stdout
		fmt.Printf("%d,%d,%v,%v,%v\n", pr.Id, len(pr.Reviewers), pr.CreatedAt.Format("2006-01-02"), pr.MergedAt.Format("2006-01-02"), cycleTime)
	}

}

func filterPRs(prs []PullRequest) []PullRequest {
	// Start with an empty slice
	filtered := []PullRequest{}

	// This will filter all PR's in August - Adjust accordingly
	for _, pr := range prs {
		if pr.MergedAt.Before(time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC)) || pr.MergedAt.After(time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC)) {
			continue
		}

		// Add the filtered PR to the `filtered` slice
		filtered = append(filtered, pr)
	}
	return filtered
}

// recursively page through the Github GraphQL endpoint
func getPrs(before string, prs []PullRequest) []PullRequest {

	// rate limiting hehe
	time.Sleep(time.Second * 2)

	// GUARD CLAUSE
	// if merged at is over 1 month ago, return the PR's
	if len(prs) > 1 {
		lastPr := prs[len(prs)-1]

		if lastPr.MergedAt.Before(time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC)) {
			return prs
		}

	}

	// if before is empty, this is the last page, so we set it to "null"
	// this is required for GraphQL
	if before == "" {
		before = "null"
	} else {
		// if the before page is not empty, we should send before, wrapped in quotes
		before = fmt.Sprintf("\"%s\"", before)
	}

	// Create a hash that holds the GraphQL query
	// We assign 'before' dynamically using a format string -- fmt.Sprintf(args)
	// We grab 100 merged PR's at a time

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

	// Serialize the GraphQL query as JSON
	jsonValue, err := json.Marshal(jsonData)

	// crash/exit/terminate if this goes wrong
	if err != nil {
		panic(err)
	}

	// Create a request to Github GraphQL endpoint via post
	req, err := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", bytes.NewBuffer(jsonValue))

	// crash/exit/terminate if this goes wrong
	if err != nil {
		panic(err)
	}

	// Add the authorization header with the user's Github token
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", *token))

	// Send the request, and assign the response to 'res'
	res, err := http.DefaultClient.Do(req)

	// crash/exit/terminate if this goes wrong
	if err != nil {
		panic(err)
	}

	// Clean up the resources when we're done
	defer res.Body.Close()

	// Read the response from the Github endpoint
	body, err := io.ReadAll(res.Body)

	// crash/exit/terminate if this goes wrong
	if err != nil {
		panic(err)
	}

	// This is where the magic happens :)
	// Create an empty variable holding a Page struct (kind of like a class)
	var page Page

	// Transform the JSON directly into an 'instance' of the class
	// If this goes wrong, we exit
	if err := json.Unmarshal(body, &page); err != nil {
		panic(err)
	}

	// Here we create the Pull Request struct, making the data easier to work with
	for _, pr := range page.Data.Repository.PullRequests.Edges {
		p := PullRequest{CreatedAt: pr.Node.CreatedAt, Id: pr.Node.Number, MergedAt: pr.Node.MergedAt, Reviewers: map[string]bool{}}
		for _, r := range pr.Node.Reviews.Edges {
			// We only want approved reviews, thank you :)
			if r.Node.State == "APPROVED" {
				// this looks a bit weird, but basically we are using a map (hash) to
				// only keep unique approvers
				p.Reviewers[r.Node.Author.Login] = true
			}
		}
		// Add the Pull Request to our slice
		prs = append(prs, p)
	}

	// Recurse ;)
	return getPrs(page.Data.Repository.PullRequests.PageInfo.StartCursor, prs)

}
