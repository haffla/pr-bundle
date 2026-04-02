package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

var (
	branchName   string
	showAll      bool
	targetBranch = "staging"
	prNumbers    []string
)

func main() {
	parseArgs()

	requireCommand("gh")
	requireGHAuth()
	requireCleanWorktree()

	if len(prNumbers) == 0 {
		prNumbers = interactiveSelect()
	}

	if len(prNumbers) == 0 {
		fmt.Println("No PRs selected.")
		return
	}

	run("git", "fetch", "origin")

	if branchName == "" {
		branchName = fmt.Sprintf("merge-prs/%s", time.Now().Format("20060102-150405"))
	}

	fmt.Printf("Creating branch '%s' from origin/main...\n", branchName)
	run("git", "checkout", "-b", branchName, "origin/main")

	var merged, skipped []string

	for _, prNum := range prNumbers {
		title, branch := prInfo(prNum)

		fmt.Printf("\nMerging PR #%s: %s (%s)...\n", prNum, title, branch)

		if err := runErr("git", "fetch", "origin", branch); err != nil {
			fmt.Printf("Warning: Could not fetch branch '%s' for PR #%s. Skipping.\n", branch, prNum)
			skipped = append(skipped, fmt.Sprintf("#%s: %s", prNum, title))
			continue
		}

		if err := runErr("git", "merge", "origin/"+branch, "--no-edit"); err != nil {
			action := handleConflict(prNum, title)
			switch action {
			case "continue":
				runErr("git", "add", "-A")
				runErr("git", "commit", "--no-edit")
				merged = append(merged, fmt.Sprintf("#%s: %s", prNum, title))
			case "skip":
				runErr("git", "merge", "--abort")
				skipped = append(skipped, fmt.Sprintf("#%s: %s", prNum, title))
			case "abort":
				runErr("git", "merge", "--abort")
				goto summary
			}
		} else {
			merged = append(merged, fmt.Sprintf("#%s: %s", prNum, title))
		}
	}

summary:
	fmt.Println("\n=== Summary ===")
	if len(merged) > 0 {
		fmt.Println("Merged:")
		for _, m := range merged {
			fmt.Println("  " + m)
		}
	}
	if len(skipped) > 0 {
		fmt.Println("Skipped:")
		for _, s := range skipped {
			fmt.Println("  " + s)
		}
	}
	fmt.Printf("Branch: %s\n", branchName)

	fmt.Printf("\nPush to origin/%s? [y/N] ", targetBranch)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if strings.EqualFold(strings.TrimSpace(scanner.Text()), "y") {
		run("git", "push", "origin", branchName+":"+targetBranch, "--force-with-lease")
		fmt.Printf("Pushed to origin/%s\n", targetBranch)
	} else {
		fmt.Println("Not pushed. You can push later with:")
		fmt.Printf("  git push origin %s:%s\n", branchName, targetBranch)
	}
}

func parseArgs() {
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--branch":
			i++
			if i >= len(args) {
				fatal("--branch requires a value")
			}
			branchName = args[i]
		case "--all":
			showAll = true
		case "--target":
			i++
			if i >= len(args) {
				fatal("--target requires a value")
			}
			targetBranch = args[i]
		case "-h", "--help":
			printUsage()
			os.Exit(0)
		default:
			if strings.HasPrefix(args[i], "-") {
				fatal("Unknown option: " + args[i])
			}
			prNumbers = append(prNumbers, args[i])
		}
	}
}

func printUsage() {
	fmt.Print(`Usage: merge-prs [OPTIONS] [PR_NUMBER ...]

Merge multiple PRs into a new branch. When no PR numbers are given,
opens an interactive picker.

Options:
  --branch NAME    Custom branch name (default: merge-prs/<timestamp>)
  --all            Show all open PRs in interactive mode (default: dependabot only)
  --target BRANCH  Remote branch for push prompt (default: staging)
  -h, --help       Show this help message

Examples:
  merge-prs 721 720 719
  merge-prs --branch staging-deps 721 720
  merge-prs                        # interactive mode
  merge-prs --all                  # interactive, all authors
`)
}

func interactiveSelect() []string {
	fmt.Println("Fetching open PRs...")

	ghArgs := []string{"pr", "list", "--state", "open", "--limit", "100",
		"--json", "number,title,author,headRefName"}
	if !showAll {
		ghArgs = append(ghArgs, "--author", "app/dependabot")
	}

	out, err := exec.Command("gh", ghArgs...).Output()
	if err != nil {
		fatal("Failed to list PRs: " + err.Error())
	}

	var prs []struct {
		Number      int    `json:"number"`
		Title       string `json:"title"`
		Author      struct{ Login string } `json:"author"`
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(out, &prs); err != nil {
		fatal("Failed to parse PR list: " + err.Error())
	}

	if len(prs) == 0 {
		fmt.Println("No open PRs found.")
		os.Exit(0)
	}

	var options []huh.Option[string]
	for _, pr := range prs {
		label := fmt.Sprintf("#%d  %s  %s", pr.Number, pr.Author.Login, pr.Title)
		options = append(options, huh.NewOption(label, strconv.Itoa(pr.Number)))
	}

	var selected []string
	err = huh.NewMultiSelect[string]().
		Title("Select PRs to merge (space to select, enter to confirm)").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil
	}

	return selected
}

func prInfo(number string) (title, branch string) {
	out, err := exec.Command("gh", "pr", "view", number,
		"--json", "title,headRefName").Output()
	if err != nil {
		fatal(fmt.Sprintf("Failed to get info for PR #%s: %s", number, err))
	}

	var pr struct {
		Title       string `json:"title"`
		HeadRefName string `json:"headRefName"`
	}
	if err := json.Unmarshal(out, &pr); err != nil {
		fatal("Failed to parse PR info: " + err.Error())
	}
	return pr.Title, pr.HeadRefName
}

func handleConflict(prNum, title string) string {
	fmt.Printf("\nConflict while merging PR #%s: %s\n", prNum, title)
	fmt.Println("Resolve the conflicts, then type 'continue', 'skip', or 'abort':")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.TrimSpace(scanner.Text()) {
		case "continue":
			return "continue"
		case "skip":
			return "skip"
		case "abort":
			return "abort"
		default:
			fmt.Println("Type 'continue', 'skip', or 'abort':")
		}
	}
	return "abort"
}

func requireCommand(name string) {
	if _, err := exec.LookPath(name); err != nil {
		fatal(name + " is not installed")
	}
}

func requireGHAuth() {
	if err := exec.Command("gh", "auth", "status").Run(); err != nil {
		fatal("Not authenticated with gh. Run 'gh auth login' first.")
	}
}

func requireCleanWorktree() {
	out, _ := exec.Command("git", "status", "--porcelain").Output()
	if len(strings.TrimSpace(string(out))) > 0 {
		fatal("Working tree is not clean. Commit or stash your changes first.")
	}
}

func run(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal(fmt.Sprintf("Command failed: %s %s", name, strings.Join(args, " ")))
	}
}

func runErr(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "Error: "+msg)
	os.Exit(1)
}
