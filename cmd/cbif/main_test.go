package main

import (
	"flag"
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
		want     bool
		wantErr  bool
	}{
		{
			name:     "success",
			envKey:   "PROJECT_ID",
			envValue: "fake-project-id",
			want:     true,
		},
		{
			name:     "fail-with-projects",
			envKey:   "PROJECT_ID",
			envValue: "fake-project-id",
			projects: flaga.Strings{
				Values:   []string{"different-project-id"},
				Assigned: true,
			},
			want:    false,
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
			want:    false,
			wantErr: true,
		},
		{
			name: "fail-with-empty",
			empty: flaga.String{
				Value:    "not-empty-value",
				Assigned: true,
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "fail-with-notempty",
			notEmpty: flaga.String{
				Value:    "", // value is empty.
				Assigned: true,
			},
			want:    false,
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
			got, err := shouldRun()
			if (err != nil) != tt.wantErr {
				t.Errorf("shouldRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("shouldRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_main(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		projects flaga.Strings
	}{
		{
			name: "success-run",
			env:  "bananas",
			projects: flaga.Strings{
				Values:   []string{"bananas"},
				Assigned: true,
			},
		},
		{
			name: "success-do-not-run",
			env:  "different-project",
			projects: flaga.Strings{
				Values:   []string{"bananas"},
				Assigned: true,
			},
		},
	}
	for _, tt := range tests {
		flag.CommandLine.Parse([]string{"echo"})

		ifProjects = tt.projects
		// Reset other global flags.
		ifBranches = flaga.Strings{}
		ifEmpty = flaga.String{}
		ifNotEmpty = flaga.String{}
		t.Run(tt.name, func(t *testing.T) {
			if tt.env != "" {
				d := osx.MustSetenv("PROJECT_ID", tt.env)
				defer d()
			}
			main()
		})
	}
}
