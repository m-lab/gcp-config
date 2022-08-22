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

	"golang.org/x/oauth2"

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
	triggerName string
	ghToken     string
)

var usage = `
NAME:
  cbctl - cloud build control, to manage and run cloud build triggers.

DESCRIPTION:

  Multi-phase development and deployment strategies require similar triggering
  rules across multiple projects. Among other things, cbctl automates creating
  "sandbox", "staging", and "production" build triggers for GitHub repos.

  NOTE: before this utility will work, you must [manually connect the Github
  repository to Cloud
  Build](https://cloud.google.com/build/docs/automating-builds/create-manage-triggers#connect_repo)
  from the Cloud Build interface. The process of connecting a repository will
  ask you if you want to create a trigger. **DO NOT** create a new trigger in
  that workflow, or you will have duplicate triggers. That

  Basic usage:

  cbctl <flags> <operation>

  Supported operations:

  * create: create a trigger in one project only
  * create-mlab-projects: creates a standard trigger in all M-Lab projects
  * details: show detailed information about every trigger in a project
  * list: a basic list of all triggers in a project
  * trigger: triggers a build

EXAMPLES:

  # Create standard build triggers across all three M-Lab projects. This
  # operation is only suitable for use by M-Lab.
  cbctl -org m-lab -repo example-repo create-mlab-projects

  # List triggers in mlab-sandbox project.
  cbctl -project mlab-sandbox list

  # Trigger a build. NOTE: in the mlab-sandbox project you _must_ specify the
  # -branch flag. In mlab-staging and mlab-oti, you have no control over the
  # build ref that gets used for the build. In mlab-staging the build ref will
  # always be HEAD of the default branch (usually "main" or "master"). In
  # mlab-oti, the build ref will always be the most recent release tag for the
  # repo. For example:
  cbctl -project mlab-sandbox -repo some-repo -branch sandbox-fix-bug trigger
  cbctl -project mlab-staging -repo some-repo trigger
  cbctl -project mlab-oti -repo some-repo trigger

USAGE:
`

func init() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
		flag.PrintDefaults()
	}
	flag.StringVar(&org, "org", "m-lab", "Github organization containing repos (e.g. m-lab)")
	flag.StringVar(&repo, "repo", "", "Github source repo (e.g. ndt-server)")
	flag.StringVar(&project, "project", "mlab-sandbox", "GCP project name")
	flag.StringVar(&filename, "filename", "cloudbuild.yaml", "Name of the cloudbuild configuration to use")
	flag.StringVar(&branch, "branch", "", "Pattern to match branches for this trigger")
	flag.StringVar(&tag, "tag", "", "Pattern to match tags for this trigger")
	flag.StringVar(&triggerName, "trigger_name", "", "Name of build trigger to use")
	flag.StringVar(&ghToken, "github_token", "", "Token for authenticating to the Github API")
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

// githubGetLatestRelease returns a *github.RepositoryRelease representing the
// most recent release for the passed repo.
func githubGetLatestRelease(ctx context.Context, gh *github.Client, org, repo string) *github.RepositoryRelease {
	r, _, err := gh.Repositories.GetLatestRelease(ctx, org, repo)
	rtx.Must(err, "Failed to get latest release for repo: %s", repo)

	return r
}

// githubGetRepository returns a *github.Repository for the passed repo name.
func githubGetRepository(ctx context.Context, gh *github.Client, org, repo string) *github.Repository {
	r, _, err := gh.Repositories.Get(ctx, org, repo)
	rtx.Must(err, "Failed to get Repository for repo: %s", repo)

	return r
}

// getBuildTargetRef returns an appropriate target reference (tag/branch) for a
// repository. In production (mlab-oti) this will always be a release tag name, in
// staging it will always be the default branch name for the repository, and in
// sandbox it will be whatever branch name was passed in via the -branch flag.
func getBuildTargetRef(ctx context.Context, gh *github.Client, project string) string {
	var ref string
	switch project {
	case "mlab-oti":
		rel := githubGetLatestRelease(ctx, gh, org, repo)
		ref = *rel.TagName
	case "mlab-staging":
		rep := githubGetRepository(ctx, gh, org, repo)
		ref = *rep.DefaultBranch
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
	rtx.Must(flagx.ArgsFromEnvWithLog(flag.CommandLine, false), "Could not parse env args")

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
		var ghClient *github.Client

		if repo == "" {
			log.Fatalln("You must specify a repo (-repo flag) when using the 'trigger' operation")
		}

		if triggerName == "" {
			triggerName = formatName(org, repo)
		}
		bt, err := cmd.Get(ctx, project, triggerName)
		rtx.Must(err, "Failed to get BuildTrigger")

		if ghToken != "" {
			tokenSource := oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: ghToken},
			)
			ghClient = github.NewClient(oauth2.NewClient(ctx, tokenSource))

		} else {
			ghClient = github.NewClient(nil)
		}

		ref := getBuildTargetRef(ctx, ghClient, project)
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

	case "create-mlab-projects":
		log.Println("Creating standard triggers across all M-Lab projects")

		opts := []struct {
			tag     string
			branch  string
			project string
		}{
			{
				branch:  "^sandbox-.*",
				project: "mlab-sandbox",
			},
			{
				branch:  "^main$",
				project: "mlab-staging",
			},
			{
				tag:     "^v([0-9.]+)+",
				project: "mlab-oti",
			},
		}
		for _, opt := range opts {
			bt := newPushTrigger(org, repo, opt.tag, opt.branch, filename)
			t, err := cmd.Create(ctx, opt.project, bt)
			rtx.Must(ignoreConflict(err), "Failed to create trigger")
			pretty.Print(t)
		}

	default:
		flag.Usage()
	}
}
