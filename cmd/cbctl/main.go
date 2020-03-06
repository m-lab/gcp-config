package main

import (
	"context"
	"fmt"

	"github.com/google/go-github/github"
	"google.golang.org/api/cloudbuild/v1"
)

var (
	project = "mlab-sandbox"
)

func main() {
	ctx := context.Background()
	s, _ := cloudbuild.NewService(ctx)
	ts := cloudbuild.NewProjectsTriggersService(s)
	list := ts.List("mlab-sandbox")

	list.Pages(ctx, func(r *cloudbuild.ListBuildTriggersResponse) error {
		for _, item := range r.Triggers {
			if item.Name == "switch-config" {

				if project == "mlab-oti" {
					client := github.NewClient(nil)
					rels, _, _ := client.Repositories.ListReleases(ctx, "m-lab", "siteinfo", &github.ListOptions{Page: 1, PerPage: 1})
					for _, rel := range rels {
						tagName := rel.GetTagName()
					}
					rc := ts.Run(project, item.Id, &cloudbuild.RepoSource{ProjectId: project, TagName: tagName})
				} else {
					rc := ts.Run(project, item.Id, &cloudbuild.RepoSource{ProjectId: project, BranchName: "master"})
				}
				op, _ := rc.Do()
				fmt.Println(op.Name)
			}
		}
		return nil
	})
}
