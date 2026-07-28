package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/black-osiris-technologies/git-fleet/internal/releaseflow"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
	"github.com/black-osiris-technologies/git-fleet/internal/semver"
	"github.com/black-osiris-technologies/git-fleet/internal/syncplan"
)

// defaultJobs is the per-repository concurrency used when the caller does not
// override it with --jobs. Repositories are independent working trees, so their
// git operations run safely in parallel.
const defaultJobs = 8

// Build metadata, stamped at release time via -ldflags. Defaults keep local
// builds honest about being unversioned.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "scan":
		return runScan(args[1:])
	case "status":
		return runStatus(args[1:])
	case "sync":
		return runSync(args[1:])
	case "release-start":
		return runReleaseStart(args[1:])
	case "release-tag":
		return runReleaseTag(args[1:])
	case "release-pr":
		return runReleasePR(args[1:])
	case "release-merge":
		return runReleaseMerge(args[1:])
	case "release-finish":
		return runReleaseFinish(args[1:])
	case "version", "--version", "-v":
		fmt.Printf("git-fleet %s (commit %s, built %s)\n", version, commit, date)
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

// repoLine is the per-repository outcome shared by the action-based commands.
type repoLine struct {
	Repo    string `json:"repo"`
	Action  string `json:"action"`
	Message string `json:"message"`
}

func runScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	paths := make([]string, len(repos))
	for i, r := range repos {
		paths[i] = r.Path
	}

	if *jsonOut {
		return encodeJSON(map[string]any{"repos": paths})
	}
	for _, path := range paths {
		fmt.Println(path)
	}
	return nil
}

type statusLine struct {
	Repo   string `json:"repo"`
	Branch string `json:"branch,omitempty"`
	Dirty  bool   `json:"dirty"`
	Error  string `json:"error,omitempty"`
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to inspect in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]statusLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		status, err := repo.Status(path)
		if err != nil {
			lines[index] = statusLine{Repo: path, Error: err.Error()}
			return
		}
		lines[index] = statusLine{Repo: path, Branch: status.Branch, Dirty: status.Dirty}
	})

	if *jsonOut {
		return encodeJSON(map[string]any{"results": lines})
	}
	for _, line := range lines {
		if line.Error != "" {
			fmt.Printf("%s\tERROR\t%s\n", line.Repo, line.Error)
			continue
		}
		fmt.Printf("%s\t%s\t%s\n", line.Repo, line.Branch, cleanLabel(line.Dirty))
	}
	return nil
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	target := fs.String("target", "develop", "target branch: develop, master, main, or latest-release")
	dryRun := fs.Bool("dry-run", false, "show the sync plan without changing repositories")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		plan := syncplan.Plan(path, *target)
		if !*dryRun {
			plan = syncplan.Execute(path, *target)
		}
		lines[index] = repoLine{Repo: plan.RepoPath, Action: string(plan.Action), Message: plan.Message}
	})

	return renderActionReport(*jsonOut, lines, []string{"ready", "done", "skipped", "failed"})
}

func runReleaseStart(args []string) error {
	fs := flag.NewFlagSet("release-start", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	major := fs.Bool("major", false, "start the next major line (default is the next minor line)")
	version := fs.String("version", "", "explicit MAJOR.MINOR[.PATCH] version; required for repositories with no release tags")
	branchFormat := fs.String("branch-format", releaseflow.DefaultBranchFormat, "branch name template using {major}, {minor}, {patch}")
	base := fs.String("base", releaseflow.DefaultBaseBranch, "integration branch to cut the release line from")
	dryRun := fs.Bool("dry-run", false, "show the release-start plan without creating branches")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := releaseflow.StartOptions{
		Bump:            semver.BumpMinor,
		BranchFormat:    *branchFormat,
		BaseBranch:      *base,
		ExplicitVersion: *version,
	}
	if *major {
		opts.Bump = semver.BumpMajor
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		result := releaseflow.PlanStart(path, opts)
		if !*dryRun {
			result = releaseflow.StartRelease(path, opts, releaseflow.RealRunner{})
		}
		lines[index] = toRepoLine(result)
	})

	return renderActionReport(*jsonOut, lines, []string{"planned", "done", "skipped", "failed"})
}

func runReleaseTag(args []string) error {
	fs := flag.NewFlagSet("release-tag", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "release branch: explicit branch name or latest-release")
	tagFormat := fs.String("tag-format", releaseflow.DefaultTagFormat, "tag name template using {major}, {minor}, {patch}")
	version := fs.String("version", "", "explicit MAJOR.MINOR.PATCH to tag instead of the next patch on the line")
	message := fs.String("message", "", "annotation message; defaults to \"Release <tag>\"")
	dryRun := fs.Bool("dry-run", false, "show the release-tag plan without creating tags")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := releaseflow.TagOptions{
		From:            *from,
		TagFormat:       *tagFormat,
		Message:         *message,
		ExplicitVersion: *version,
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		result := releaseflow.PlanTag(path, opts)
		if !*dryRun {
			result = releaseflow.CreateTag(path, opts, releaseflow.RealRunner{})
		}
		lines[index] = toRepoLine(result)
	})

	return renderActionReport(*jsonOut, lines, []string{"planned", "done", "skipped", "failed"})
}

func runReleasePR(args []string) error {
	fs := flag.NewFlagSet("release-pr", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "source branch: explicit branch name or latest-release")
	to := fs.String("to", "master", "target branch, usually master or develop")
	dryRun := fs.Bool("dry-run", false, "show the release PR plan without creating pull requests")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		result := releaseflow.PlanPR(path, *from, *to)
		if !*dryRun {
			result = releaseflow.CreatePR(path, *from, *to, releaseflow.RealRunner{})
		}
		lines[index] = toRepoLine(result)
	})

	return renderActionReport(*jsonOut, lines, []string{"planned", "done", "skipped", "failed"})
}

func runReleaseMerge(args []string) error {
	fs := flag.NewFlagSet("release-merge", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "source branch: explicit branch name or latest-release")
	to := fs.String("to", "master", "target branch, usually master or develop")
	mergeMethod := fs.String("merge-method", "merge", "GitHub merge method; only merge is supported")
	dryRun := fs.Bool("dry-run", false, "show the release merge plan without merging pull requests")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mergeMethod != "merge" {
		return fmt.Errorf("unsupported merge method %q; only merge commits are allowed", *mergeMethod)
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		result := releaseflow.PlanMerge(path, *from, *to)
		if !*dryRun {
			result = releaseflow.MergePR(path, *from, *to, releaseflow.RealRunner{})
		}
		lines[index] = toRepoLine(result)
	})

	return renderActionReport(*jsonOut, lines, []string{"planned", "done", "skipped", "failed"})
}

func runReleaseFinish(args []string) error {
	fs := flag.NewFlagSet("release-finish", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "release branch: explicit branch name or latest-release")
	master := fs.String("master", "master", "production branch to merge the release into")
	develop := fs.String("develop", "develop", "integration branch to merge the release into")
	deleteBranch := fs.Bool("delete-branch", false, "delete the release branch on origin after both merges succeed")
	dryRun := fs.Bool("dry-run", false, "show the release-finish plan without merging or deleting")
	jobs := fs.Int("jobs", defaultJobs, "number of repositories to process in parallel")
	jsonOut := fs.Bool("json", false, "emit machine-readable JSON instead of text")
	if err := fs.Parse(args); err != nil {
		return err
	}

	opts := releaseflow.FinishOptions{
		From:          *from,
		MasterBranch:  *master,
		DevelopBranch: *develop,
		DeleteBranch:  *deleteBranch,
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	lines := make([]repoLine, len(repos))
	mapRepos(repos, *jobs, func(index int, path string) {
		result := releaseflow.PlanFinish(path, opts)
		if !*dryRun {
			result = releaseflow.FinishRelease(path, opts, releaseflow.RealRunner{})
		}
		lines[index] = toRepoLine(result)
	})

	return renderActionReport(*jsonOut, lines, []string{"planned", "done", "skipped", "failed"})
}

// mapRepos runs work over repos with up to jobs concurrent workers, writing each
// result into the slice position matching the repository's index so output stays
// deterministic regardless of completion order.
func mapRepos(repos []repo.Repository, jobs int, work func(index int, path string)) {
	if jobs < 1 {
		jobs = 1
	}
	var wg sync.WaitGroup
	slots := make(chan struct{}, jobs)
	for i, r := range repos {
		wg.Add(1)
		slots <- struct{}{}
		go func(index int, path string) {
			defer wg.Done()
			defer func() { <-slots }()
			work(index, path)
		}(i, r.Path)
	}
	wg.Wait()
}

func toRepoLine(result releaseflow.Result) repoLine {
	return repoLine{Repo: result.RepoPath, Action: string(result.Action), Message: result.Message}
}

// renderActionReport prints the per-repository lines and a summary, either as
// tab-separated text or as JSON. labels are the summary counters to report, in
// order, matching the possible action values (lowercased).
func renderActionReport(jsonOut bool, lines []repoLine, labels []string) error {
	if jsonOut {
		summary := map[string]int{"total": len(lines)}
		for _, label := range labels {
			summary[label] = countAction(lines, label)
		}
		return encodeJSON(map[string]any{"results": lines, "summary": summary})
	}

	for _, line := range lines {
		fmt.Printf("%s\t%s\t%s\n", line.Repo, line.Action, line.Message)
	}
	fmt.Printf("summary\ttotal=%d", len(lines))
	for _, label := range labels {
		fmt.Printf("\t%s=%d", label, countAction(lines, label))
	}
	fmt.Println()
	return nil
}

func countAction(lines []repoLine, label string) int {
	count := 0
	for _, line := range lines {
		if strings.EqualFold(line.Action, label) {
			count++
		}
	}
	return count
}

func encodeJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func cleanLabel(dirty bool) string {
	if dirty {
		return "dirty"
	}
	return "clean"
}

func printUsage() {
	fmt.Println("Usage: git-fleet <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan    Discover Git repositories")
	fmt.Println("  status  Show branch and dirty state")
	fmt.Println("  sync    Plan or execute safe repository synchronization")
	fmt.Println("  release-start  Plan or create the next release branch from origin/develop")
	fmt.Println("  release-tag    Plan or cut the next patch tag on a release line")
	fmt.Println("  release-pr     Plan or create release pull requests")
	fmt.Println("  release-merge  Plan or merge release pull requests with merge commits")
	fmt.Println("  release-finish Plan or merge a release into master and develop, then optionally delete it")
	fmt.Println("  version Print the git-fleet version")
	fmt.Println()
	fmt.Println("Common options: --json (machine-readable output), --jobs N (parallelism)")
}
