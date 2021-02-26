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

// Package cloudbuild wraps the upstream cloudbuild package in a simplified
// interface. All code MUST be correct by inspection.
package cbctl

import (
	"context"

	"google.golang.org/api/cloudbuild/v1"
)

// Trigger wraps the cloudbuild API into a simple interface.
type Trigger struct {
	service *cloudbuild.Service
}

// NewTrigger creates a new Trigger instance.
func NewTrigger(s *cloudbuild.Service) *Trigger {
	return &Trigger{
		service: s,
	}
}

func (t *Trigger) Run(ctx context.Context, project string, triggerId string, src *cloudbuild.RepoSource) (*cloudbuild.Operation, error) {
	return t.service.Projects.Triggers.Run(project, triggerId, src).Context(ctx).Do()
}

func (t *Trigger) Create(ctx context.Context, project string, bt *cloudbuild.BuildTrigger) (*cloudbuild.BuildTrigger, error) {
	return t.service.Projects.Triggers.Create(project, bt).Context(ctx).Do()
}

func (t *Trigger) List(ctx context.Context, project string, visit func(tr *cloudbuild.ListBuildTriggersResponse) error) error {
	c := t.service.Projects.Triggers.List(project)
	return c.Pages(ctx, visit)
}
