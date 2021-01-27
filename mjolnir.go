package main

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/v32/github"
)

// create type for cross repository issue
type crossRepoIssue struct {
	owner string
	repositoryName string
	issueNumber int
}

var (
	globalFixesIssueRE = regexp.MustCompile(`(?i)(?:close|closes|closed|fix|fixes|fixed|resolve|resolves|resolved)((?:[\s]+#[\d]+)(?:[\s,]+#[\d]+)*(?:[\n\r\s,]|$))`)
	fixesIssueRE       = regexp.MustCompile(`[\s,]+#`)
	cleanNumberRE      = regexp.MustCompile(`[\n\r\s,]`)
)

// closeRelatedIssues Closes issues listed in the PR description.
func closeRelatedIssues(ctx context.Context, client *github.Client, owner string, repositoryName string, pr *github.PullRequest, dryRun bool) error {
	issueNumbers := parseIssueFixes(pr.GetBody())

	// log.Printf("Natch - issue numbers: %v", issueNumbers)
	log.Printf("Natch - owner: %s, repo: %s, pr: %s", owner, repositoryName, pr.GetBody())

	for _, issueNumber := range issueNumbers {
		log.Printf("PR #%d: closes issue #%d, add milestones %s", pr.GetNumber(), issueNumber, pr.Milestone.GetTitle())
		if !dryRun {
			err := closeIssue(ctx, client, owner, repositoryName, pr, issueNumber)
			if err != nil {
				return fmt.Errorf("unable to close issue #%d: %w", issueNumber, err)
			}
		}

		// Add comment if needed
		if pr.Base.GetRef() != "master" {
			message := fmt.Sprintf("Closed by #%d.", pr.GetNumber())

			log.Printf("PR #%d: issue #%d, add comment: %s", pr.GetNumber(), issueNumber, message)

			if !dryRun {
				err := addComment(ctx, client, owner, repositoryName, issueNumber, message)
				if err != nil {
					return fmt.Errorf("unable to add comment on issue #%d: %w", issueNumber, err)
				}
			}
		}
	}

	crossRepoIssues := parseCrossRepoIssueFixes(pr.GetBody())

	for _, crossRepoIssue := range crossRepoIssues {
		log.Printf("PR #%d: closes %s/%s issue #%d, add milestones %s", pr.GetNumber(), crossRepoIssue.owner, crossRepoIssue.repositoryName, crossRepoIssue.issueNumber, pr.Milestone.GetTitle())

		if !dryRun {
			err := closeIssue(ctx, client, crossRepoIssue.owner, crossRepoIssue.repositoryName, pr, crossRepoIssue.issueNumber)
			if err != nil {
				return fmt.Errorf("unable to close %s/%s issue #%d: %w", crossRepoIssue.owner, crossRepoIssue.repositoryName, issueNumber, err)
			}
		}
	}

	return nil
}

func closeIssue(ctx context.Context, client *github.Client, owner string, repositoryName string, pr *github.PullRequest, issueNumber int) error {
	var milestone *int
	if pr.Milestone != nil {
		milestone = pr.Milestone.Number
	}

	issueRequest := &github.IssueRequest{
		Milestone: milestone,
		State:     github.String("closed"),
	}

	_, _, err := client.Issues.Edit(ctx, owner, repositoryName, issueNumber, issueRequest)
	return err
}

func addComment(ctx context.Context, client *github.Client, owner string, repositoryName string, issueNumber int, message string) error {
	issueComment := &github.IssueComment{
		Body: github.String(message),
	}
	_, _, err := client.Issues.CreateComment(ctx, owner, repositoryName, issueNumber, issueComment)
	return err
}

func parseIssueFixes(text string) []int {
	var issueNumbers []int

	submatch := globalFixesIssueRE.FindStringSubmatch(strings.ReplaceAll(text, ":", ""))

	if len(submatch) != 0 {
		issuesRaw := fixesIssueRE.Split(submatch[1], -1)

		for _, issueRaw := range issuesRaw {
			cleanIssueRaw := cleanNumberRE.ReplaceAllString(issueRaw, "")
			if len(cleanIssueRaw) != 0 {
				numb, err := strconv.ParseInt(cleanIssueRaw, 10, 16)
				if err != nil {
					log.Println(err)
				}
				issueNumbers = append(issueNumbers, int(numb))
			}
		}
	}
	return issueNumbers
}

func parseCrossRepoIssueFixes(text string) []crossRepoIssue {
	log.Printf("parseCrossRepoIssueFixes fuc - %s", text)
	crossRepoIssues := []crossRepoIssue{
		{
			owner: "knatch",
			repositoryName: "knatch.github.io",
			issueNumber: 1,
		},
	}
	return crossRepoIssues
}