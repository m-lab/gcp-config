package main

import (
	"os"
	"testing"

	"github.com/m-lab/gcp-config/flaga"
	"github.com/m-lab/go/osx"
)

func Test_shouldRun(t *testing.T) {
	tests := []struct {
		name     string
		projects flaga.Strings
		branches flaga.Strings
		empty    flaga.String
		notEmpty flaga.String
		envKey   string
		envValue string
		wantErr  bool
	}{
		{
			name:     "success",
			envKey:   "PROJECT_ID",
			envValue: "fake-project-id",
		},
		{
			name:     "fail-with-projects",
			envKey:   "PROJECT_ID",
			envValue: "fake-project-id",
			projects: flaga.Strings{
				Values:   []string{"different-project-id"},
				Assigned: true,
			},
			wantErr: true,
		},
		{
			name:     "fail-with-branches",
			envKey:   "BRANCH_NAME",
			envValue: "fake-branch",
			branches: flaga.Strings{
				Values:   []string{"different-branch"},
				Assigned: true,
			},
			wantErr: true,
		},
		{
			name: "fail-with-empty",
			empty: flaga.String{
				Value:    "not-empty-value",
				Assigned: true,
			},
			wantErr: true,
		},
		{
			name: "fail-with-notempty",
			notEmpty: flaga.String{
				Value:    "", // value is empty.
				Assigned: true,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ifProjects = tt.projects
			ifBranches = tt.branches
			ifNotEmpty = tt.notEmpty
			ifEmpty = tt.empty
			if tt.envKey != "" {
				d := osx.MustSetenv(tt.envKey, tt.envValue)
				defer d()
			}
			err := shouldRun()
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_main(t *testing.T) {
	orig := os.Args
	tests := []struct {
		name     string
		projEnv  string
		projects flaga.Strings
		args     []string
		code     int
	}{
		{
			name:    "command-runs",
			projEnv: "bananas",
			projects: flaga.Strings{
				Values:   []string{"bananas"},
				Assigned: true,
			},
			args: []string{"echo"},
			code: 0,
		},
		{
			name:    "command-does-not-run",
			projEnv: "different-project",
			projects: flaga.Strings{
				Values:   []string{"bananas"},
				Assigned: true,
			},
			args: []string{"echo"},
			code: 0,
		},
		{
			name:    "command-runs-and-exists-non-zero",
			projEnv: "bananas",
			projects: flaga.Strings{
				Values:   []string{"bananas"},
				Assigned: true,
			},
			args: []string{"false"},
			code: 1,
		},
	}
	for _, tt := range tests {
		// Update os.Args for main's call to flag.Parse().
		os.Args = append(orig, tt.args...)

		// Use project flags and reset the other global flags.
		ifProjects = tt.projects
		ifBranches = flaga.Strings{}
		ifEmpty = flaga.String{}
		ifNotEmpty = flaga.String{}

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

			main()

			if code != tt.code {
				t.Errorf("main() wrong exit code; got %d, want %d", code, tt.code)
			}
		})
	}
}
