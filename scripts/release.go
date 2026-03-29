package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ReleaseRequest struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish,omitempty"`
	Name            string `json:"name"`
	Body            string `json:"body"`
	Draft           bool   `json:"draft"`
	Prerelease      bool   `json:"prerelease"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: release <github-token>")
		os.Exit(1)
	}

	token := os.Args[1]
	owner := "mailbus"
	repo := "mailbus"
	tag := "v0.1.0"

	// Read release notes
	notes, err := os.ReadFile("RELEASE_NOTES.md")
	if err != nil {
		fmt.Printf("Error reading RELEASE_NOTES.md: %v\n", err)
		os.Exit(1)
	}

	// Create release request
	release := ReleaseRequest{
		TagName:    tag,
		Name:       "MailBus v0.1.0 - Email-based Message Bus",
		Body:       string(notes),
		Draft:      false,
		Prerelease: true,
	}

	jsonData, err := json.Marshal(release)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	// Create release via GitHub API
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error creating request: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		fmt.Printf("Release created successfully!\n")
		fmt.Printf("Status: %d\n", resp.StatusCode)
		var result map[string]interface{}
		json.Unmarshal(body, &result)
		if htmlURL, ok := result["html_url"].(string); ok {
			fmt.Printf("URL: %s\n", htmlURL)
		}
	} else {
		fmt.Printf("Error creating release!\n")
		fmt.Printf("Status: %d\n", resp.StatusCode)
		fmt.Printf("Response: %s\n", string(body))
		os.Exit(1)
	}
}
