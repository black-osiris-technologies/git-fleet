package githubapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"
)

const apiVersion = "2022-11-28"

type Client struct {
	owner      string
	repo       string
	apiBaseURL string
	token      string
	httpClient *http.Client
}

func NewForRepo(repoPath string) (*Client, error) {
	remote, err := originURL(repoPath)
	if err != nil {
		return nil, err
	}

	host, owner, repository, apiBaseURL, err := parseRemote(remote)
	if err != nil {
		return nil, err
	}

	token := tokenFromEnv(host)
	if token == "" {
		if strings.EqualFold(host, "github.com") {
			return nil, fmt.Errorf("GitHub API authentication required: set GH_TOKEN or GITHUB_TOKEN")
		}
		return nil, fmt.Errorf("GitHub API authentication required for %s: set GH_ENTERPRISE_TOKEN or GITHUB_ENTERPRISE_TOKEN", host)
	}

	return &Client{
		owner:      owner,
		repo:       repository,
		apiBaseURL: apiBaseURL,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) CreatePullRequest(base, head, title, body string) (string, bool, error) {
	_, found, err := c.FindOpenPullRequest(base, head)
	if err != nil {
		return "", false, err
	}
	if found {
		return "", false, nil
	}

	payload := map[string]any{
		"title": title,
		"head":  head,
		"base":  base,
		"body":  body,
	}
	var response struct {
		HTMLURL string `json:"html_url"`
	}
	status, responseBody, err := c.doJSON(http.MethodPost, c.repoPath("pulls"), nil, payload, &response)
	if err != nil {
		return "", false, err
	}
	if status == http.StatusUnprocessableEntity && pullRequestAlreadyExists(responseBody) {
		return "", false, nil
	}
	if status != http.StatusCreated {
		return "", false, apiError(http.MethodPost, c.repoPath("pulls"), status, responseBody)
	}
	return response.HTMLURL, true, nil
}

func (c *Client) FindOpenPullRequest(base, head string) (int, bool, error) {
	query := url.Values{}
	query.Set("state", "open")
	query.Set("base", base)
	query.Set("head", c.owner+":"+head)
	query.Set("per_page", "1")

	var response []struct {
		Number int `json:"number"`
	}
	status, responseBody, err := c.doJSON(http.MethodGet, c.repoPath("pulls"), query, nil, &response)
	if err != nil {
		return 0, false, err
	}
	if status != http.StatusOK {
		return 0, false, apiError(http.MethodGet, c.repoPath("pulls"), status, responseBody)
	}
	if len(response) == 0 {
		return 0, false, nil
	}
	return response[0].Number, true, nil
}

func (c *Client) MergePullRequest(number int) (string, error) {
	endpoint := c.repoPath("pulls", strconv.Itoa(number), "merge")
	payload := map[string]any{"merge_method": "merge"}
	var response struct {
		Merged  bool   `json:"merged"`
		Message string `json:"message"`
	}
	status, responseBody, err := c.doJSON(http.MethodPut, endpoint, nil, payload, &response)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", apiError(http.MethodPut, endpoint, status, responseBody)
	}
	if !response.Merged {
		if response.Message == "" {
			response.Message = "GitHub did not merge the pull request"
		}
		return "", fmt.Errorf("%s", response.Message)
	}
	if response.Message != "" {
		return response.Message, nil
	}
	return fmt.Sprintf("merged pull request #%d", number), nil
}

func (c *Client) doJSON(method, endpoint string, query url.Values, payload any, out any) (int, []byte, error) {
	requestURL := strings.TrimRight(c.apiBaseURL, "/") + endpoint
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, nil, fmt.Errorf("encode GitHub API request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequest(method, requestURL, body)
	if err != nil {
		return 0, nil, fmt.Errorf("build GitHub API request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("X-GitHub-Api-Version", apiVersion)
	req.Header.Set("User-Agent", "git-fleet")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("GitHub API %s %s: %w", method, endpoint, err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return 0, nil, fmt.Errorf("read GitHub API response: %w", err)
	}
	if out != nil && len(responseBody) > 0 && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if err := json.Unmarshal(responseBody, out); err != nil {
			return 0, nil, fmt.Errorf("decode GitHub API response: %w", err)
		}
	}
	return resp.StatusCode, responseBody, nil
}

func (c *Client) repoPath(parts ...string) string {
	values := []string{"repos", c.owner, c.repo}
	values = append(values, parts...)
	return "/" + path.Join(values...)
}

func originURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return "", fmt.Errorf("git remote get-url origin: %s", message)
	}
	remote := strings.TrimSpace(string(output))
	if remote == "" {
		return "", fmt.Errorf("origin remote URL is empty")
	}
	return remote, nil
}

func parseRemote(remote string) (host, owner, repository, apiBaseURL string, err error) {
	remote = strings.TrimSpace(remote)
	if remote == "" {
		return "", "", "", "", fmt.Errorf("origin remote URL is empty")
	}

	var repoPath string
	apiScheme := "https"
	apiAuthority := ""
	if strings.Contains(remote, "://") {
		parsed, parseErr := url.Parse(remote)
		if parseErr != nil || parsed.Hostname() == "" {
			return "", "", "", "", fmt.Errorf("unsupported origin remote URL %q", remote)
		}
		host = parsed.Hostname()
		repoPath = strings.TrimPrefix(parsed.Path, "/")
		if parsed.Scheme == "http" {
			return "", "", "", "", fmt.Errorf("insecure HTTP origin %q is not supported for authenticated GitHub API operations", remote)
		}
		if parsed.Scheme == "https" {
			// Preserve an explicit HTTPS port for GitHub Enterprise API requests.
			// SSH transport ports are not assumed to be API ports.
			apiScheme = "https"
			apiAuthority = parsed.Host
		}
	} else if at := strings.LastIndex(remote, "@"); at >= 0 {
		hostAndPath := remote[at+1:]
		colon := strings.Index(hostAndPath, ":")
		if colon <= 0 {
			return "", "", "", "", fmt.Errorf("unsupported origin remote URL %q", remote)
		}
		host = hostAndPath[:colon]
		repoPath = hostAndPath[colon+1:]
	} else {
		return "", "", "", "", fmt.Errorf("origin %q is not a GitHub SSH/HTTP remote", remote)
	}

	repoPath = strings.TrimSuffix(strings.Trim(repoPath, "/"), ".git")
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", "", fmt.Errorf("cannot determine GitHub owner/repository from origin %q", remote)
	}
	owner, repository = parts[0], parts[1]

	if strings.EqualFold(host, "github.com") {
		apiBaseURL = "https://api.github.com"
	} else {
		if apiAuthority == "" {
			apiAuthority = host
		}
		apiBaseURL = apiScheme + "://" + apiAuthority + "/api/v3"
	}
	return host, owner, repository, apiBaseURL, nil
}

func tokenFromEnv(host string) string {
	names := []string{"GH_TOKEN", "GITHUB_TOKEN"}
	if !strings.EqualFold(host, "github.com") {
		// Never fall back to a GitHub.com token for an arbitrary origin host.
		// Enterprise hosts require an explicitly enterprise-scoped credential.
		names = []string{"GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}
	}
	for _, name := range names {
		if token := strings.TrimSpace(os.Getenv(name)); token != "" {
			return token
		}
	}
	return ""
}

func pullRequestAlreadyExists(body []byte) bool {
	var response struct {
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &response) != nil {
		return false
	}
	if strings.Contains(strings.ToLower(response.Message), "pull request already exists") {
		return true
	}
	for _, item := range response.Errors {
		if strings.Contains(strings.ToLower(item.Message), "pull request already exists") {
			return true
		}
	}
	return false
}

func apiError(method, endpoint string, status int, body []byte) error {
	message := strings.TrimSpace(http.StatusText(status))
	var response struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &response) == nil && strings.TrimSpace(response.Message) != "" {
		message = strings.TrimSpace(response.Message)
	}
	if message == "" {
		message = "unexpected response"
	}
	return fmt.Errorf("GitHub API %s %s: HTTP %d: %s", method, endpoint, status, message)
}
