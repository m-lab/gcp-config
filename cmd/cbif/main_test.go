package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/osx"
	"github.com/m-lab/go/rtx"
	"gopkg.in/m-lab/pipe.v3"
)

func Test_main(t *testing.T) {
	// Create a tempdir as a fake /workspace and link target location.
	tmpdir, err := ioutil.TempDir("", "maintesting-")
	rtx.Must(err, "failed to create tempdir")
	defer os.RemoveAll(tmpdir)

	// Unpack the fake git data to use during tests.
	b, err := pipe.CombinedOutput(
		pipe.Exec("tar", "-C", tmpdir, "-xf", "../../testdata/fake.git.tar.gz"),
	)
	fmt.Println(string(b))

	tests := []struct {
		name string
		env  map[string]string
		args []string
		code int
	}{
		{
			name: "command-runs-using-env-correct-branch",
			env: map[string]string{
				"BRANCH_NAME": "correct-branch",
				"PROJECT_ID":  "correct-project",
				"BRANCH_IN":   "correct-branch",
				"PROJECT_IN":  "correct-project",
			},
			args: []string{"fake-cbif", "echo"},
			code: 0,
		},
		{
			name: "command-does-not-run-wrong-branch",
			env: map[string]string{
				"BRANCH_NAME": "wrong-branch",
			},
			args: []string{"fake-cbif", "-branch-in=correct-branch", "echo"},
			code: 0,
		},
		{
			name: "command-does-not-run-wrong-project",
			env: map[string]string{
				"PROJECT_ID": "wrong-project",
			},
			args: []string{"fake-cbif", "-project-in=current-project", "echo"},
			code: 0,
		},
		{
			name: "command-runs-and-exists-non-zero",
			args: []string{"fake-cbif", "false"},
			code: 1,
		},
		{
			name: "setup-workspace-link-run-success",
			args: []string{"fake-cbif", "pwd"},
			env: map[string]string{
				"WORKSPACE":      tmpdir,
				"WORKSPACE_LINK": path.Join(tmpdir, "go/this/is/a/path"),
			},
		},
		{
			name: "setup-git-directory",
			args: []string{"fake-cbif", "stat", ".git"},
			env: map[string]string{
				"WORKSPACE":      tmpdir,
				"WORKSPACE_LINK": path.Join(tmpdir, "go/this/is/a/path"),
				"GIT_ORIGIN_URL": tmpdir + "/fake.git",
				"COMMIT_SHA":     "2a5f2af7fad0af0e1a354f42660a78420cd4751f",
				"SINGLE_COMMAND": "true",
			},
		},
	}
	for _, tt := range tests {
		// Update os.Args for main's call to flag.Parse().
		os.Args = tt.args

		// Reset the other global flags.
		projects = flagx.StringArray{}
		branches = flagx.StringArray{}

		// Completely reset command line flags.
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		setupFlags()

		// Save exit code.
		code := 0
		osExit = func(c int) {
			code = c
		}

		t.Run(tt.name, func(t *testing.T) {
			for e, v := range tt.env {
				d := osx.MustSetenv(e, v)
				defer d()
			}

			main()

			if code != tt.code {
				t.Errorf("main() wrong exit code; got %d, want %d", code, tt.code)
			}
		})
	}
}
