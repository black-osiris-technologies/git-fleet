package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/black-osiris-technologies/git-fleet/internal/releaseflow"
	"github.com/black-osiris-technologies/git-fleet/internal/repo"
	"github.com/black-osiris-technologies/git-fleet/internal/semver"
	"github.com/black-osiris-technologies/git-fleet/internal/syncplan"
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
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runScan(args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	for _, r := range repos {
		fmt.Println(r.Path)
	}
	return nil
}

func runStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	for _, r := range repos {
		status, err := repo.Status(r.Path)
		if err != nil {
			fmt.Printf("%s\tERROR\t%v\n", r.Path, err)
			continue
		}
		fmt.Printf("%s\t%s\t%s\n", r.Path, status.Branch, cleanLabel(status.Dirty))
	}
	return nil
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	target := fs.String("target", "develop", "target branch: develop, master, main, or latest-release")
	dryRun := fs.Bool("dry-run", false, "show the sync plan without changing repositories")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	summary := syncplan.Summary{}
	for _, r := range repos {
		plan := syncplan.Plan(r.Path, *target)
		if !*dryRun {
			plan = syncplan.Execute(r.Path, *target)
		}
		summary.Add(plan)
		fmt.Printf("%s\t%s\t%s\n", plan.RepoPath, plan.Action, plan.Message)
	}

	fmt.Printf("summary\ttotal=%d\tready=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Ready, summary.Done, summary.Skipped, summary.Failed)
	return nil
}

func runReleaseStart(args []string) error {
	fs := flag.NewFlagSet("release-start", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	major := fs.Bool("major", false, "start the next major line (default is the next minor line)")
	version := fs.String("version", "", "explicit MAJOR.MINOR[.PATCH] version; required for repositories with no release tags")
	branchFormat := fs.String("branch-format", releaseflow.DefaultBranchFormat, "branch name template using {major}, {minor}, {patch}")
	base := fs.String("base", releaseflow.DefaultBaseBranch, "integration branch to cut the release line from")
	dryRun := fs.Bool("dry-run", false, "show the release-start plan without creating branches")
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

	summary := releaseflow.Summary{}
	for _, r := range repos {
		result := releaseflow.PlanStart(r.Path, opts)
		if !*dryRun {
			result = releaseflow.StartRelease(r.Path, opts, releaseflow.RealRunner{})
		}
		summary.Add(result)
		fmt.Printf("%s\t%s\t%s\n", result.RepoPath, result.Action, result.Message)
	}
	fmt.Printf("summary\ttotal=%d\tplanned=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Planned, summary.Done, summary.Skipped, summary.Failed)
	return nil
}

func runReleaseTag(args []string) error {
	fs := flag.NewFlagSet("release-tag", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "release branch: explicit branch name or latest-release")
	tagFormat := fs.String("tag-format", releaseflow.DefaultTagFormat, "tag name template using {major}, {minor}, {patch}")
	version := fs.String("version", "", "explicit MAJOR.MINOR.PATCH to tag instead of the next patch on the line")
	message := fs.String("message", "", "annotation message; defaults to \"Release <tag>\"")
	dryRun := fs.Bool("dry-run", false, "show the release-tag plan without creating tags")
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

	summary := releaseflow.Summary{}
	for _, r := range repos {
		result := releaseflow.PlanTag(r.Path, opts)
		if !*dryRun {
			result = releaseflow.CreateTag(r.Path, opts, releaseflow.RealRunner{})
		}
		summary.Add(result)
		fmt.Printf("%s\t%s\t%s\n", result.RepoPath, result.Action, result.Message)
	}
	fmt.Printf("summary\ttotal=%d\tplanned=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Planned, summary.Done, summary.Skipped, summary.Failed)
	return nil
}

func runReleasePR(args []string) error {
	fs := flag.NewFlagSet("release-pr", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "source branch: explicit branch name or latest-release")
	to := fs.String("to", "master", "target branch, usually master or develop")
	dryRun := fs.Bool("dry-run", false, "show the release PR plan without creating pull requests")
	if err := fs.Parse(args); err != nil {
		return err
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	summary := releaseflow.Summary{}
	for _, r := range repos {
		result := releaseflow.PlanPR(r.Path, *from, *to)
		if !*dryRun {
			result = releaseflow.CreatePR(r.Path, *from, *to, releaseflow.RealRunner{})
		}
		summary.Add(result)
		fmt.Printf("%s\t%s\t%s\n", result.RepoPath, result.Action, result.Message)
	}
	fmt.Printf("summary\ttotal=%d\tplanned=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Planned, summary.Done, summary.Skipped, summary.Failed)
	return nil
}

func runReleaseMerge(args []string) error {
	fs := flag.NewFlagSet("release-merge", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "source branch: explicit branch name or latest-release")
	to := fs.String("to", "master", "target branch, usually master or develop")
	mergeMethod := fs.String("merge-method", "merge", "GitHub merge method; only merge is supported")
	dryRun := fs.Bool("dry-run", false, "show the release merge plan without merging pull requests")
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

	summary := releaseflow.Summary{}
	for _, r := range repos {
		result := releaseflow.PlanMerge(r.Path, *from, *to)
		if !*dryRun {
			result = releaseflow.MergePR(r.Path, *from, *to, releaseflow.RealRunner{})
		}
		summary.Add(result)
		fmt.Printf("%s\t%s\t%s\n", result.RepoPath, result.Action, result.Message)
	}
	fmt.Printf("summary\ttotal=%d\tplanned=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Planned, summary.Done, summary.Skipped, summary.Failed)
	return nil
}

func runReleaseFinish(args []string) error {
	fs := flag.NewFlagSet("release-finish", flag.ContinueOnError)
	root := fs.String("root", ".", "root directory to scan")
	from := fs.String("from", "latest-release", "release branch: explicit branch name or latest-release")
	master := fs.String("master", "master", "production branch to merge the release into")
	develop := fs.String("develop", "develop", "integration branch to merge the release into")
	deleteBranch := fs.Bool("delete-branch", false, "delete the release branch on origin after both merges succeed")
	dryRun := fs.Bool("dry-run", false, "show the release-finish plan without merging or deleting")
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

	summary := releaseflow.Summary{}
	for _, r := range repos {
		result := releaseflow.PlanFinish(r.Path, opts)
		if !*dryRun {
			result = releaseflow.FinishRelease(r.Path, opts, releaseflow.RealRunner{})
		}
		summary.Add(result)
		fmt.Printf("%s\t%s\t%s\n", result.RepoPath, result.Action, result.Message)
	}
	fmt.Printf("summary\ttotal=%d\tplanned=%d\tdone=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Planned, summary.Done, summary.Skipped, summary.Failed)
	return nil
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
}
