// Copyright © 2021 gcp-config Authors.
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
	"net/http"
	"os"
	"time"

	"github.com/m-lab/gcp-config/internal/cbctl"

	"github.com/google/go-github/v35/github"
	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"
	"github.com/stephen-soltesz/pretty"

	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	org         string
	repo        string
	project     string
	filename    string
	branch      string
	tag         string
	projects    flagx.StringArray
	triggerName string
)

var usage = `
NAME:
  cbctl - cloud build control, to manage cloud build triggers.

DESCRIPTION:

  Multi-phase development and deployment strategies require similar
  triggering rules across multiple projects. cbctl automates creating
  "sandbox", "staging", and "production" build triggers for GitHub repos.

  NOTE: you must manually add cloud build GitHub integration to a new
  repository. *DO NOT* use that workflow to create a new trigger, or you will
  have duplicate triggers.

EXAMPLES:

  # Create standard build triggers across three projects.
  cbctl -org m-lab -repo example-repo \
      -projects mlab-sandbox,mlab-staging,mlab-oti create-projects

  # List triggers in mlab-sandbox project.
  cbctl -project mlab-sandbox list

USAGE:
`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&org, "org", "m-lab", "Github organization containing repos (e.g. m-lab)")
	flag.StringVar(&repo, "repo", "", "Github source repo (e.g. ndt-server)")
	flag.StringVar(&project, "project", "mlab-sandbox", "GCP project name")
	flag.Var(&projects, "projects", "A sequence of GCP project names")
	flag.StringVar(&filename, "filename", "cloudbuild.yaml", "Name of the cloudbuild configuration to use")
	flag.StringVar(&branch, "branch", "", "Pattern to match branches for this trigger")
	flag.StringVar(&tag, "tag", "", "Pattern to match tags for this trigger")
	flag.StringVar(&triggerName, "trigger_name", "", "Name of build trigger to use")
}

func mustArg(n int) string {
	args := flag.Args()
	if len(args)-1 < n {
		flag.Usage()
		os.Exit(1)
	}
	return args[n]
}

func formatDesc(tag, branch string) string {
	var d string
	switch {
	case tag != "":
		d = "Tag matching " + tag
	case branch != "":
		d = "Push to branch matching " + branch
	default:
		panic("not yet supported")
	}
	return d
}

func formatName(org, repo string) string {
	return "push-" + org + "-" + repo + "-trigger"
}

func newPushTrigger(org, repo, tag, branch, filename string) *cloudbuild.BuildTrigger {
	var name string
	if triggerName != "" {
		name = triggerName
	} else {
		name = formatName(org, repo)
	}
	bt := &cloudbuild.BuildTrigger{
		// NOTE: trigger name depends only on the repo, so multiple projects use the same name.
		Name:        name,
		Description: formatDesc(tag, branch),
		Filename:    filename,
		Github: &cloudbuild.GitHubEventsConfig{
			Name:  repo,
			Owner: org,
			Push: &cloudbuild.PushFilter{
				Tag:    tag,
				Branch: branch,
			},
		},
	}
	return bt
}

// A conflict error is returned when the build trigger already exists. We can
// ignore this case.
func ignoreConflict(err error) error {
	if e, ok := err.(*googleapi.Error); ok {
		if e.Code == http.StatusConflict {
			return nil
		}
		pretty.Print(e)
	}
	return err
}

func eventDesc(g *cloudbuild.GitHubEventsConfig) string {
	if g.Push.Branch != "" {
		return "Push to branch"
	}
	if g.Push.Tag != "" {
		return "Push new tag"
	}
	return "Unknown"
}

// githubGetLatestRelease returns the latest release tag for a Github
// repository.
func githubGetLatestRelease(gh *cbctl.Github, org, repo string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	r, _, err := gh.Client.Repositories.GetLatestRelease(ctx, org, repo)
	rtx.Must(err, "Failed to get latest release for repo: %s", repo)

	return *r.TagName
}

// githubGetDefaultBranch returns the name of the default branch for a Github
// repository.
func githubGetDefaultBranch(gh *cbctl.Github, org, repo string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	r, _, err := gh.Client.Repositories.Get(ctx, org, repo)
	rtx.Must(err, "Failed to get default branch for repo: %s", repo)

	return *r.DefaultBranch
}

// getBuildTargetRef returns an appropriate target reference (tag/branch) for a
// repository. In production (mlab-oti) this will always be a release tag name, in
// staging it will always be the default branch name for the repository, and in
// sandbox it will be whatever branch name was passed in via the -branch flag.
func getBuildTargetRef(gh *cbctl.Github, project string) string {
	var ref string
	switch project {
	case "mlab-oti":
		ref = githubGetLatestRelease(gh, org, repo)
	case "mlab-staging":
		ref = githubGetDefaultBranch(gh, org, repo)
	default:
		if branch == "" {
			log.Fatalf("Branch must be specified when triggering a build in project: %s", project)
		}
		ref = branch
	}
	return ref
}

func main() {
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s, err := cloudbuild.NewService(ctx, option.WithScopes(cloudbuild.CloudPlatformScope))
	rtx.Must(err, "Failed to create cb service")

	cmd := cbctl.NewTrigger(s)

	// Assert that only one of tag or branch are not empty.
	if tag != "" && branch != "" {
		log.Println("Specify only one of -branch or -tag")
		os.Exit(1)
	}

	op := mustArg(0)
	switch op {
	case "list":
		// short description.
		fmt.Printf("%-5s %-40s %-45s %s\n", "Enabled", "Name", "Description", "Filename")
		err := cmd.List(ctx, project, func(tr *cloudbuild.ListBuildTriggersResponse) error {
			for _, t := range tr.Triggers {
				fmt.Printf("%-5t %-40s %-45q %s\n",
					!t.Disabled,
					t.Name, t.Description,
					t.Filename,
				)
			}
			return nil
		})
		rtx.Must(err, "Failed to list triggers")
	case "details":
		// detailed description.
		err := cmd.List(ctx, project, func(tr *cloudbuild.ListBuildTriggersResponse) error {
			for _, t := range tr.Triggers {
				pretty.Print(t)
			}
			return nil
		})
		rtx.Must(err, "Failed to list triggers")

	case "trigger":
		var rs *cloudbuild.RepoSource

		if repo == "" {
			log.Fatalln("You must specify a repo (-repo flag) when using the 'trigger' operation")
		}

		if triggerName == "" {
			triggerName = formatName(org, repo)
		}
		bt, err := cmd.Get(ctx, project, triggerName)
		rtx.Must(err, "Failed to get BuildTrigger")

		c := github.NewClient(nil)
		gh := cbctl.NewGithub(c)

		ref := getBuildTargetRef(gh, project)
		if project == "mlab-oti" {
			rs = &cloudbuild.RepoSource{
				ProjectId: project,
				RepoName:  repo,
				TagName:   ref,
			}
			log.Printf("Triggering build for repo %s on tag %s in project %s", repo, ref, project)
		} else {
			rs = &cloudbuild.RepoSource{
				ProjectId:  project,
				RepoName:   repo,
				BranchName: ref,
			}
			log.Printf("Triggering build for repo %s on branch %s in project %s", repo, ref, project)
		}

		_, err = cmd.Run(ctx, project, bt.Id, rs)
		rtx.Must(err, "Failed to run build trigger for repo '%s' with repository build target '%s' in project '%s'", repo, ref, project)

	case "create":
		log.Println("Creating single trigger")
		bt := newPushTrigger(org, repo, tag, branch, filename)
		t, err := cmd.Create(ctx, project, bt)
		rtx.Must(err, "Failed to create trigger")
		pretty.Print(t)

	case "create-projects":
		log.Println("Creating standard triggers across several projects")

		opts := []struct {
			tag    string
			branch string
		}{
			{branch: "sandbox-.*"},
			{branch: "main"},
			{tag: "v([0-9.]+)+"},
		}
		for i, opt := range opts {
			if len(projects) < len(opts) {
				log.Printf("Skipping: %#v", opt)
				continue
			}
			bt := newPushTrigger(org, repo, opt.tag, opt.branch, filename)
			t, err := cmd.Create(ctx, projects[i], bt)
			rtx.Must(ignoreConflict(err), "Failed to create trigger")
			pretty.Print(t)
		}

	default:
		flag.Usage()
	}
}
