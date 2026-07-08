package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/black-osiris-technologies/omp-git-fleet/internal/repo"
	"github.com/black-osiris-technologies/omp-git-fleet/internal/syncplan"
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
	if !*dryRun {
		return fmt.Errorf("sync execution is not implemented yet; use --dry-run")
	}

	repos, err := repo.Discover(*root)
	if err != nil {
		return err
	}

	summary := syncplan.Summary{}
	for _, r := range repos {
		plan := syncplan.Plan(r.Path, *target)
		summary.Add(plan)
		fmt.Printf("%s\t%s\t%s\n", plan.RepoPath, plan.Action, plan.Message)
	}

	fmt.Printf("summary\ttotal=%d\tready=%d\tskipped=%d\tfailed=%d\n", summary.Total, summary.Ready, summary.Skipped, summary.Failed)
	return nil
}

func cleanLabel(dirty bool) string {
	if dirty {
		return "dirty"
	}
	return "clean"
}

func printUsage() {
	fmt.Println("Usage: omp-git-fleet <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  scan    Discover Git repositories")
	fmt.Println("  status  Show branch and dirty state")
	fmt.Println("  sync    Plan safe repository synchronization")
}
