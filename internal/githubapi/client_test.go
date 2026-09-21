package githubapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseRemote(t *testing.T) {
	tests := []struct {
		remote string
		host   string
		owner  string
		repo   string
		api    string
	}{
		{"git@github.com:black-osiris-technologies/git-fleet.git", "github.com", "black-osiris-technologies", "git-fleet", "https://api.github.com"},
		{"https://github.com/black-osiris-technologies/git-fleet.git", "github.com", "black-osiris-technologies", "git-fleet", "https://api.github.com"},
		{"ssh://git@github.example.com/owner/repo.git", "github.example.com", "owner", "repo", "https://github.example.com/api/v3"},
		{"https://github.example.com:8443/owner/repo.git", "github.example.com", "owner", "repo", "https://github.example.com:8443/api/v3"},
	}
	for _, tt := range tests {
		t.Run(tt.remote, func(t *testing.T) {
			host, owner, repo, api, err := parseRemote(tt.remote)
			if err != nil {
				t.Fatalf("parseRemote() error = %v", err)
			}
			if host != tt.host || owner != tt.owner || repo != tt.repo || api != tt.api {
				t.Fatalf("parseRemote() = %q %q %q %q", host, owner, repo, api)
			}
		})
	}
}

func TestCreatePullRequestUsesNativeAPI(t *testing.T) {
	var calls []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.String())
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/pulls":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, "[]")
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/pulls":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), "\"head\":\"release-1.4\"") || !strings.Contains(string(body), "\"base\":\"master\"") {
				t.Fatalf("unexpected body: %s", body)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, "{\"html_url\":\"https://github.example.test/owner/repo/pull/42\"}")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := &Client{owner: "owner", repo: "repo", apiBaseURL: server.URL, token: "test-token", httpClient: server.Client()}
	url, created, err := client.CreatePullRequest("master", "release-1.4", "Release", "body")
	if err != nil {
		t.Fatalf("CreatePullRequest() error = %v", err)
	}
	if !created || url != "https://github.example.test/owner/repo/pull/42" {
		t.Fatalf("CreatePullRequest() = %q %v", url, created)
	}
	if len(calls) != 2 {
		t.Fatalf("calls = %#v, want list + create", calls)
	}
}

func TestCreatePullRequestSkipsExistingOpenPR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected mutation %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "[{\"number\":7}]")
	}))
	defer server.Close()

	client := &Client{owner: "owner", repo: "repo", apiBaseURL: server.URL, token: "test-token", httpClient: server.Client()}
	_, created, err := client.CreatePullRequest("master", "release-1.4", "Release", "body")
	if err != nil {
		t.Fatalf("CreatePullRequest() error = %v", err)
	}
	if created {
		t.Fatal("CreatePullRequest() created duplicate PR")
	}
}

func TestMergePullRequestForcesMergeCommitMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/repos/owner/repo/pulls/42/merge" {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "\"merge_method\":\"merge\"") {
			t.Fatalf("unexpected body: %s", body)
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, "{\"merged\":true,\"message\":\"Pull Request successfully merged\"}")
	}))
	defer server.Close()

	client := &Client{owner: "owner", repo: "repo", apiBaseURL: server.URL, token: "test-token", httpClient: server.Client()}
	message, err := client.MergePullRequest(42)
	if err != nil {
		t.Fatalf("MergePullRequest() error = %v", err)
	}
	if message != "Pull Request successfully merged" {
		t.Fatalf("MergePullRequest() = %q", message)
	}
}


func TestEnterpriseHostDoesNotUseGenericGitHubToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "github-dot-com-token")
	t.Setenv("GITHUB_TOKEN", "github-dot-com-token-2")
	t.Setenv("GH_ENTERPRISE_TOKEN", "")
	t.Setenv("GITHUB_ENTERPRISE_TOKEN", "")

	if got := tokenFromEnv("ghe.example.com"); got != "" {
		t.Fatalf("tokenFromEnv(enterprise) = %q, want no generic-token fallback", got)
	}

	t.Setenv("GH_ENTERPRISE_TOKEN", "enterprise-token")
	if got := tokenFromEnv("ghe.example.com"); got != "enterprise-token" {
		t.Fatalf("tokenFromEnv(enterprise) = %q, want enterprise token", got)
	}
}

func TestGitHubDotComUsesGenericToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "github-dot-com-token")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_ENTERPRISE_TOKEN", "enterprise-token")

	if got := tokenFromEnv("github.com"); got != "github-dot-com-token" {
		t.Fatalf("tokenFromEnv(github.com) = %q, want GH_TOKEN", got)
	}
}
