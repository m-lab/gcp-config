package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/pretty"
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

*/

var (
	ignoreErrors   bool
	commandTimeout time.Duration
	projects       flagx.StringArray
	branches       flagx.StringArray
)

func init() {
	flag.BoolVar(&ignoreErrors, "ignore-errors", false, "Ignore non-zero exit codes when executing commands.")
	flag.DurationVar(&commandTimeout, "command-timeout", time.Hour, "Individual time out for each command to complete.")
	flag.Var(&projects, "project-in", "Run if the current project is one of the conditional projects.")
	flag.Var(&branches, "branch-in", "Run if the current branch is one of the conditional branches.")
}

func createCmd(ctx context.Context, args []string, sout, serr *os.File) *exec.Cmd {
	log.Println("run:", args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = sout
	cmd.Stderr = serr
	return cmd
}

var osExit = os.Exit

func checkExit(err error, ps *os.ProcessState) {
	if err == nil {
		log.Printf("success: pid:%d code:%d\n", ps.Pid(), ps.ExitCode())
		return
	}
	log.Printf("error: pid:%d code:%d err:%s\n", ps.Pid(), ps.ExitCode(), err.Error())
	if !ignoreErrors {
		osExit(ps.ExitCode())
	}
}

func shouldRun(flags foundFlags) (string, bool) {
	project := os.Getenv("PROJECT_ID")
	if flags.Assigned("PROJECT_IN") && !projects.Contains(project) {
		return fmt.Sprintf("RUN:false PROJECT_IN=%v does not include current project (%s)\n",
			projects, project), false
	}
	branch := os.Getenv("BRANCH_NAME")
	if flags.Assigned("BRANCH_IN") && !branches.Contains(branch) {
		return fmt.Sprintf("RUN:false BRANCH_IN=%v does not include current branch (%s)\n",
			branches, branch), false
	}

	reason := "RUN:true"
	if flags.Assigned("PROJECT_IN") {
		reason += fmt.Sprintf(" AND PROJECT_IN=%v contains %q", projects, project)
	}
	if flags.Assigned("BRANCH_IN") {
		reason += fmt.Sprintf(" AND BRANCH_IN=%v contains %q", branches, branch)
	}
	return reason, true
}

// foundFlags tracks whether flags were found during flag parsing.
type foundFlags map[string]struct{}

func (f foundFlags) Assigned(k string) bool {
	_, found := f[k]
	return found
}

func asEnvNames(original map[string]struct{}) foundFlags {
	assigned := make(map[string]struct{})
	for k := range original {
		assigned[flagx.MakeShellVariableName(k)] = struct{}{}
	}
	return assigned
}

func main() {
	flag.Parse()
	rtx.Must(flagx.ArgsFromEnv(flag.CommandLine), "Failed to parse flags")
	v := asEnvNames(flagx.AssignedFlags(flag.CommandLine))
	pretty.Print(os.Args)
	pretty.Print(v)
	reason, run := shouldRun(v)
	log.Println(reason)
	if !run {
		osExit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	args := flag.CommandLine.Args()
	if len(args) > 0 {
		cmd := createCmd(ctx, args, os.Stdout, os.Stderr)
		err := cmd.Run()
		checkExit(err, cmd.ProcessState)
	}
}
