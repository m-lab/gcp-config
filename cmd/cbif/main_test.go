package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/osx"
)

func Test_main(t *testing.T) {
	orig := os.Args
	tests := []struct {
		name      string
		projEnv   string
		branchEnv string
		args      []string
		code      int
	}{
		{
			name: "command-runs-by-default",
			args: []string{"echo"},
			code: 0,
		},
		{
			name:      "command-runs-correct-branch",
			branchEnv: "correct-branch",
			projEnv:   "correct-project",
			args:      []string{"-branch-in=correct-branch", "-project-in=correct-project", "echo"},
			code:      0,
		},
		{
			name:      "command-does-not-run-wrong-branch",
			branchEnv: "wrong-branch",
			projEnv:   "correct-project",
			args:      []string{"-branch-in=correct-branch", "-project-in=correct-project", "echo"},
			code:      0,
		},
		{
			name:    "command-does-not-run-wrong-project",
			projEnv: "wrong-project",
			args:    []string{"-project-in=current-project", "echo"},
			code:    0,
		},
		{
			name:    "command-runs-and-exists-non-zero",
			projEnv: "current",
			args:    []string{"-project-in=current", "false"},
			code:    1,
		},
	}
	for _, tt := range tests {
		// Update os.Args for main's call to flag.Parse().
		os.Args = append(orig, tt.args...)
		fmt.Println("ARG:", os.Args)

		// Reset the other global flags.
		projects = flagx.StringArray{}
		branches = flagx.StringArray{}

		// Save exit code.
		code := 0
		osExit = func(c int) {
			code = c
		}

		t.Run(tt.name, func(t *testing.T) {
			if tt.projEnv != "" {
				d := osx.MustSetenv("PROJECT_ID", tt.projEnv)
				defer d()
			}
			if tt.branchEnv != "" {
				d := osx.MustSetenv("BRANCH_NAME", tt.branchEnv)
				defer d()
			}

			main()

			if code != tt.code {
				t.Errorf("main() wrong exit code; got %d, want %d", code, tt.code)
			}
		})
	}
}
