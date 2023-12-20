package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const (
	baseURL = "https://api.github.com"
)

var (
	accessToken string
	date        string
)

func init() {
	flag.StringVar(&accessToken, "token", "", "GitHub personal access token")
	flag.StringVar(&date, "date", "", "Date in YYYY-MM-DD format for which you want to retrieve GitHub activity")
	flag.Parse()

	if accessToken == "" || date == "" {
		fmt.Println("Please provide a GitHub personal access token and date.")
		flag.PrintDefaults()
		fmt.Println("Example: go run main.go -token YOUR_ACCESS_TOKEN -date 2023-01-01")
		fmt.Println("Get your personal access token here: https://github.com/settings/tokens")
		fmt.Println("Date should be in the format YYYY-MM-DD.")
		fmt.Println("Note: GitHub API requests are subject to rate limits without authentication.")
		fmt.Println("For more information, refer to: https://docs.github.com/en/rest/overview/resources-in-the-rest-api#rate-limiting")
	}
}

func main() {
	// Parse date string
	parsedDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return
	}

	// Generate URL for events API
	url := fmt.Sprintf("%s/users/%s/events", baseURL, getAuthenticatedUser())
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add authentication header
	request.Header.Set("Authorization", "token "+accessToken)

	// Send request
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return
	}
	defer response.Body.Close()

	// Read response body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Parse JSON response
	var events []map[string]interface{}
	err = json.Unmarshal(body, &events)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Display events for the specified date in markdown format
	fmt.Printf("## GitHub activity for %s\n", date)
	fmt.Println("")

	for _, event := range events {
		createdAt, ok := event["created_at"].(string)
		if !ok {
			continue
		}

		eventDate, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			fmt.Println("Error parsing event date:", err)
			continue
		}

		if eventDate.Day() == parsedDate.Day() && eventDate.Month() == parsedDate.Month() && eventDate.Year() == parsedDate.Year() {
			repo, ok := event["repo"].(map[string]interface{})
			if !ok {
				fmt.Println("Error getting repo information.")
				continue
			}

			repoName, ok := repo["name"].(string)
			if !ok {
				fmt.Println("Error getting repo name.")
				continue
			}

			eventType, ok := event["type"].(string)
			if !ok {
				fmt.Println("Error getting event type.")
				continue
			}

			var eventURL string

			switch eventType {
			case "PullRequestEvent":
				prNumber, ok := event["payload"].(map[string]interface{})["pull_request"].(map[string]interface{})["number"].(float64)
				if !ok {
					fmt.Println("Error getting PR number.")
					continue
				}
				eventURL = fmt.Sprintf("https://github.com/%s/pull/%d", repoName, int(prNumber))
			case "IssuesEvent":
				issueNumber, ok := event["payload"].(map[string]interface{})["issue"].(map[string]interface{})["number"].(float64)
				if !ok {
					fmt.Println("Error getting issue number.")
					continue
				}
				eventURL = fmt.Sprintf("https://github.com/%s/issues/%d", repoName, int(issueNumber))
			case "IssueCommentEvent":
				commentID, ok := event["payload"].(map[string]interface{})["comment"].(map[string]interface{})["id"].(float64)
				if !ok {
					fmt.Println("Error getting comment ID.")
					continue
				}

				// Check if it's a PR or Issue comment
				if issue, ok := event["payload"].(map[string]interface{})["issue"].(map[string]interface{}); ok {
					issueNumber, ok := issue["number"].(float64)
					if !ok {
						fmt.Println("Error getting issue number.")
						continue
					}
					eventURL = fmt.Sprintf("https://github.com/%s/issues/%d#issuecomment-%d", repoName, int(issueNumber), int(commentID))
				} else if pr, ok := event["payload"].(map[string]interface{})["pull_request"].(map[string]interface{}); ok {
					prNumber, ok := pr["number"].(float64)
					if !ok {
						fmt.Println("Error getting PR number.")
						continue
					}
					eventURL = fmt.Sprintf("https://github.com/%s/pull/%d#issuecomment-%d", repoName, int(prNumber), int(commentID))
				} else {
					fmt.Println("Error getting issue or PR information for comment.")
					continue
				}
			default:
				// Default to repo URL
				eventURL = fmt.Sprintf("https://github.com/%s", repoName)
			}

			fmt.Printf("- %s - [%s](%s)\n", eventType, repoName, eventURL)
		}
	}
}

func getAuthenticatedUser() string {
	url := fmt.Sprintf("%s/user", baseURL)
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	request.Header.Set("Authorization", "token "+accessToken)

	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Error sending request:", err)
		return ""
	}
	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ""
	}

	var user map[string]interface{}
	err = json.Unmarshal(body, &user)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return ""
	}

	return user["login"].(string)
}
