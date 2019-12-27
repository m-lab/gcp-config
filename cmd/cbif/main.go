package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/m-lab/gcp-config/flaga"
	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"
)

/*

Variables avilable from CB.

All builds:
  $PROJECT_ID    - build.ProjectId
  $BUILD_ID      - build.BuildId

Triggered builds:
  $REPO_NAME     - build.Source.RepoSource.RepoName
  $BRANCH_NAME   - build.Source.RepoSource.Revision.BranchName
  $TAG_NAME      - build.Source.RepoSource.Revision.TagName
  $COMMIT_SHA    - build.SourceProvenance.ResolvedRepoSource.Revision.CommitSha
  $SHORT_SHA     - The first seven characters of COMMIT_SHA
  $REVISION_ID   - $COMMIT_SHA

PR builds:
  _HEAD_BRANCH   - head branch of the pull request
  _BASE_BRANCH   - base branch of the pull request
  _HEAD_REPO_URL - url of the head repo of the pull request
  _PR_NUMBER     - number of the pull request

Variables recognized by cbif:
  IF_PROJECTS=a,b,c
  IF_BRANCHES=a,b,c
  IF_EMPTY=<value>
  IF_NOT_EMPTY=<value>
  IGNORE_ERRORS=bool

*/

var (
	ignoreErrors   bool
	commandTimeout time.Duration
	ifProjects     flaga.Strings
	ifBranches     flaga.Strings
	ifNotEmpty     flaga.String
	ifEmpty        flaga.String
)

func init() {
	flag.BoolVar(&ignoreErrors, "ignore-errors", false, "Ignore non-zero exit codes when executing commands.")
	flag.DurationVar(&commandTimeout, "command-timeout", time.Hour, "Individual time out for each command to complete.")
	flag.Var(&ifProjects, "if-projects", "Run if the current project is one of the conditional projects.")
	flag.Var(&ifBranches, "if-branches", "Run if the current branch is one of the conditional branches.")
	flag.Var(&ifNotEmpty, "if-not-empty", "Run if the given value is not empty.")
	flag.Var(&ifEmpty, "if-empty", "Run if the given value is empty.")
}

func createCmd(ctx context.Context, args []string, sout, serr *os.File) *exec.Cmd {
	log.Println("run:", args)
	if len(args) == 0 {
		args = []string{"echo"}
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = sout
	cmd.Stderr = serr
	return cmd
}

func checkExit(cmd *exec.Cmd, err error) {
	if cmd == nil || cmd.ProcessState == nil {
		log.Fatalf("Failed to run commands: %s", err)
	}
	ps := cmd.ProcessState
	if err != nil {
		log.Printf("error: pid:%d code:%d err:%s\n", ps.Pid(), ps.ExitCode(), err.Error())
		if !ignoreErrors {
			os.Exit(ps.ExitCode())
		}
	} else {
		log.Printf("success: pid:%d code:%d\n", ps.Pid(), ps.ExitCode())
	}
}

func shouldRun() (bool, error) {
	project := os.Getenv("PROJECT_ID")
	if ifProjects.Assigned && !ifProjects.Contains(project) {
		err := fmt.Errorf("SKIP: Current project (%s) does not match projects values (%v)",
			project, ifProjects.Values)
		return false, err
	}
	branch := os.Getenv("BRANCH_NAME")
	if ifBranches.Assigned && !ifBranches.Contains(branch) {
		err := fmt.Errorf("SKIP: Current branch (%s) does not match branch values (%v)",
			project, ifBranches.Values)
		return false, err
	}
	if ifEmpty.Assigned && ifEmpty.Value != "" {
		err := fmt.Errorf("SKIP: if-empty value was not empty: %q", ifEmpty.Value)
		return false, err
	}
	if ifNotEmpty.Assigned && ifNotEmpty.Value == "" {
		err := fmt.Errorf("SKIP: if-not-empty value was empty")
		return false, err
	}
	// Default to true.
	return true, nil
}

func main() {
	flag.Parse()
	rtx.Must(flagx.ArgsFromEnv(flag.CommandLine), "Failed to parse flags")

	if run, reason := shouldRun(); !run {
		// Log reason and exit without error.
		log.Println(reason)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	args := flag.CommandLine.Args()
	cmd := createCmd(ctx, args, os.Stdout, os.Stderr)
	checkExit(cmd, cmd.Run())
}
