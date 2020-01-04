package main

import (
	"flag"
	"os"
	"testing"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/osx"
)

func Test_main(t *testing.T) {
	// orig := os.Args
	tests := []struct {
		name      string
		projEnv   string
		branchEnv string
		env       map[string]string
		args      []string
		code      int
	}{
		{
			name:      "command-runs-using-env-correct-branch",
			branchEnv: "correct-branch",
			projEnv:   "correct-project",
			env: map[string]string{
				"BRANCH_IN":  "correct-branch",
				"PROJECT_IN": "correct-project",
			},
			args: []string{"fake-cbif", "echo", "env"},
			code: 0,
		},
		{
			name:      "command-does-not-run-wrong-branch",
			branchEnv: "wrong-branch",
			args:      []string{"fake-cbif", "-branch-in=correct-branch", "echo"},
			code:      0,
		},
		{
			name:    "command-does-not-run-wrong-project",
			projEnv: "wrong-project",
			args:    []string{"fake-cbif", "-project-in=current-project", "echo"},
			code:    0,
		},
		{
			name: "command-runs-and-exists-non-zero",
			args: []string{"fake-cbif", "false"},
			code: 1,
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
