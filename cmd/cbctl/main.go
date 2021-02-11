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
	"net/http"
	"os"
	"time"

	"github.com/m-lab/gcp-config/internal/cbctl"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"
	"github.com/stephen-soltesz/pretty"

	"google.golang.org/api/cloudbuild/v1"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

var (
	org      string
	repo     string
	project  string
	filename string
	branch   string
	tag      string
	projects flagx.StringArray
)

func init() {
	flag.StringVar(&org, "org", "m-lab", "Github organization containing repos (e.g. m-lab)")
	flag.StringVar(&repo, "repo", "", "Github source repo (e.g. ndt-server)")
	flag.StringVar(&project, "project", "mlab-sandbox", "GCP project name")
	flag.Var(&projects, "projects", "A sequence of GCP project names")
	flag.StringVar(&filename, "filename", "cloudbuild.yaml", "Name of the cloudbuild configuration to use")
	flag.StringVar(&branch, "branch", "", "Pattern to match branches for this trigger")
	flag.StringVar(&tag, "tag", "", "Pattern to match tags for this trigger")
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

func newPushTrigger(org, repo, tag, branch, filename string) *cloudbuild.BuildTrigger {
	bt := &cloudbuild.BuildTrigger{
		// NOTE: trigger name depends only on the repo, so multiple projects use the same name.
		Name:        "push-" + org + "-" + repo + "-trigger",
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

func newPRTrigger(org, repo, branch, filename string) *cloudbuild.BuildTrigger {
	bt := &cloudbuild.BuildTrigger{
		Name:        "pr-" + org + "-" + repo + "-trigger",
		Description: formatDesc("", branch),
		Filename:    filename,
		Github: &cloudbuild.GitHubEventsConfig{
			Name:  repo,
			Owner: org,
			PullRequest: &cloudbuild.PullRequestFilter{
				Branch: branch,
			},
		},
	}
	return bt
}

func ignore409(err error) error {
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
		/*
			Name
			Description
			Repository
			Event
			Revision filter
			Build configuration
			Status
		*/
		err := cmd.List(ctx, project, func(tr *cloudbuild.ListBuildTriggersResponse) error {
			for _, t := range tr.Triggers {
				fmt.Println(
					t.Name, t.Description, // t.Build.Source.RepoSource.RepoName, // t.Build.,
					eventDesc(t.Github), t.Filename, t.Disabled,
				)
				// pretty.Print(t)
			}
			return nil
		})
		rtx.Must(err, "Failed to list triggers")
	case "details":
		log.Println("Listing detailed build triggers")
		err := cmd.List(ctx, project, func(tr *cloudbuild.ListBuildTriggersResponse) error {
			for _, t := range tr.Triggers {
				pretty.Print(t)
			}
			return nil
		})
		rtx.Must(err, "Failed to list triggers")

	case "trigger":
		log.Println("TODO: implement trigger")

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
			{branch: "master"},
			{tag: "v([0-9.]+)+"},
		}
		for i, opt := range opts {
			if len(projects) < len(opts) {
				log.Printf("Skipping: %#v", opt)
				continue
			}
			bt := newPushTrigger(org, repo, opt.tag, opt.branch, filename)
			t, err := cmd.Create(ctx, projects[i], bt)
			rtx.Must(ignore409(err), "Failed to create trigger")
			pretty.Print(t)
		}

	default:
		flag.Usage()
	}
}
