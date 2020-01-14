// Copyright © 2019 gcp-config Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/google/shlex"
	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/logx"
	"github.com/m-lab/go/rtx"
	"gopkg.in/m-lab/pipe.v3"
)

var (
	ignoreErrors   bool
	commandTimeout time.Duration
	singleCmd      bool
	workspace      string
	workspaceLink  string
	gitOriginURL   string
	commitSha      string

	projects flagx.StringArray
	branches flagx.StringArray
)

func init() {
	setupFlags()
}

func setupFlags() {
	flag.BoolVar(&singleCmd, "single-command", false, "Run each argument as an individual command.")
	flag.BoolVar(&ignoreErrors, "ignore-errors", false, "Ignore non-zero exit codes when executing commands.")
	flag.DurationVar(&commandTimeout, "command-timeout", time.Hour, "Individual time out for each command to complete.")

	flag.Var(&projects, "project-in", "Run if the current project is one of the conditional projects.")
	flag.Var(&branches, "branch-in", "Run if the current branch is one of the conditional branches.")

	flag.StringVar(&workspaceLink, "workspace-link", "", "Absolute path to link to the /workspace directory and set PWD to linked directory")
	flag.StringVar(&gitOriginURL, "git-origin-url", "", "Git origin URL suitable for cloning")
	flag.StringVar(&commitSha, "commit-sha", "", "Commit SHA of the git commit for the current build.")
	flag.StringVar(&workspace, "workspace", "/workspace", "Source workspace directory to link into $GOPATH/src/$PROJECT_ROOT")
}

func createCmd(ctx context.Context, args []string, sout, serr *os.File) *exec.Cmd {
	log.Println("Command:", args)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stdout = sout
	cmd.Stderr = serr
	return cmd
}

func mustSplitCmd(command string) []string {
	args, err := shlex.Split(command)
	rtx.Must(err, "Failed to split command: %q", command)
	return args
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

func continueOrExitZero(flags foundFlags) {
	reason, run := shouldRun(flags)
	log.Println(reason)
	if !run {
		osExit(0)
	}
}

func shouldRun(flags foundFlags) (string, bool) {
	project := os.Getenv("PROJECT_ID")
	if flags.Assigned("PROJECT_IN") && !projects.Contains(project) {
		return fmt.Sprintf("RUN:false PROJECT_IN=%v does not include current project (%s)",
			projects, project), false
	}
	branch := os.Getenv("BRANCH_NAME")
	if flags.Assigned("BRANCH_IN") && !branches.Contains(branch) {
		return fmt.Sprintf("RUN:false BRANCH_IN=%v does not include current branch (%s)",
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

// assignedFlags discovers the set of flags specified either directly on the
// command line or indirectly through the environment.
func assignedFlags(fs *flag.FlagSet) foundFlags {
	assigned := make(map[string]struct{})
	// Assignments from the command line.
	fs.Visit(func(f *flag.Flag) {
		logx.Debug.Println("FOUND-FLAG:", flagx.MakeShellVariableName(f.Name))
		assigned[flagx.MakeShellVariableName(f.Name)] = struct{}{}
	})
	// Assignments from the environment.
	fs.VisitAll(func(f *flag.Flag) {
		envVarName := flagx.MakeShellVariableName(f.Name)
		if val, ok := os.LookupEnv(envVarName); ok {
			logx.Debug.Println("FOUND-ENV :", envVarName, val)
			assigned[envVarName] = struct{}{}
		}
	})
	return assigned
}

func trySetupGit(flags foundFlags) {
	_, gitErr := os.Stat(".git")
	if gitErr != nil && (flags.Assigned("GIT_ORIGIN_URL") && flags.Assigned("COMMIT_SHA")) {
		// Setup the .git repo if it's missing and we have the necessary info.
		rtx.Must(createGit(gitOriginURL, commitSha), "Failed to create .git directory")
	}
}

func trySetupWorkspaceLink(flags foundFlags) {
	if flags.Assigned("WORKSPACE_LINK") {
		// The process cwd maintained by the Linux kernel is the real, physical
		// path. We want processes to execute with a cwd within the symbolically
		// linked directory. Most shells manage PWD / OLDPWD in environment
		// variables independent of the kernel and libc. The Go os.Getwd() follows
		// this convention, by returning PWD if found, or using Getwd syscall
		// otherwise. By setting PWD, we allow processes that use this convention to
		// use the symlinked directory as the current working directory.
		rtx.Must(os.Setenv("PWD", mustLinkWorkspace(workspaceLink)), "Failed to set PWD")
	}
}

var pipeCombinedOutput = pipe.CombinedOutput

func createGit(originURL, sha string) error {
	b, err := pipeCombinedOutput(
		pipe.Script("# Creating .git from "+originURL,
			pipe.Exec("git", "init"),
			pipe.Exec("git", "remote", "add", "origin", originURL),
			pipe.Exec("git", "fetch", "--depth=1", "origin", sha),
			pipe.Exec("git", "reset", "--hard", "FETCH_HEAD"),
		),
	)
	fmt.Println(string(b))
	return err
}

func mustLinkWorkspace(absProjPath string) string {
	// Setup symlink.
	rtx.Must(os.MkdirAll(path.Dir(absProjPath), 0777), "Failed to make dirs: %q", path.Dir(absProjPath))
	os.Remove(absProjPath) // Remove last element of absProjPath if present. Ignore error if not.
	rtx.Must(os.Symlink(workspace, absProjPath), "Failed to create symlink: %q -> %q", absProjPath, workspace)
	log.Printf("SUCCESS! Created symlink: ln -s %q %q", workspace, absProjPath)
	rtx.Must(os.Chdir(absProjPath), "Failed to change dir to linked path: %q", absProjPath)
	return absProjPath
}

func prepareCommands(args []string) [][]string {
	commands := [][]string{}
	if singleCmd {
		if len(args) > 0 {
			commands = append(commands, args)
		}
	} else {
		for _, arg := range args {
			commands = append(commands, mustSplitCmd(arg))
		}
	}
	return commands
}

func main() {
	flag.Parse()
	rtx.Must(flagx.ArgsFromEnv(flag.CommandLine), "Failed to parse flags")

	flags := assignedFlags(flag.CommandLine)
	continueOrExitZero(flags)
	trySetupGit(flags)
	trySetupWorkspaceLink(flags)

	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	commands := prepareCommands(flag.CommandLine.Args())
	for _, command := range commands {
		cmd := createCmd(ctx, command, os.Stdout, os.Stderr)
		err := cmd.Run()
		checkExit(err, cmd.ProcessState)
	}
}
