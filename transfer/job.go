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

// Package transfer wraps the storagetransfer package in a simplified interface.
// All code MUST be correct by inspection.
package transfer

import (
	"context"
	"encoding/json"

	"google.golang.org/api/storagetransfer/v1"
)

// Job wraps the storagetransfer API into a simple interface.
type Job struct {
	service *storagetransfer.Service
	project string
}

// NewJob creates a new Job.
func NewJob(project string, service *storagetransfer.Service) *Job {
	return &Job{
		project: project,
		service: service,
	}
}

// NOTE: the storagetransfer api requires a json object as the filter string. (o_O)
// This `filter` object automates the formatting of that filter string.
type filter struct {
	Project  string   `json:"project_id"`
	Statuses []string `json:"job_statuses,omitempty"`
	Names    []string `json:"job_names,omitempty"`
}

// Jobs calls `visit` on all ENABLED transfer jobs in the current project.
func (j *Job) Jobs(ctx context.Context, visit func(resp *storagetransfer.ListTransferJobsResponse) error) error {
	f := filter{
		Project:  j.project,
		Statuses: []string{"ENABLED"},
	}
	bfilter, _ := json.Marshal(&f)
	list := j.service.TransferJobs.List(string(bfilter)).PageSize(20)
	return list.Pages(ctx, visit)
}

// Create will create a new transfer job.
func (j *Job) Create(ctx context.Context, create *storagetransfer.TransferJob) (*storagetransfer.TransferJob, error) {
	return j.service.TransferJobs.Create(create).Context(ctx).Do()
}

// Get retrieves the named transfer job.
func (j *Job) Get(ctx context.Context, name string) (*storagetransfer.TransferJob, error) {
	return j.service.TransferJobs.Get(name, j.project).Context(ctx).Do()
}

// Update updates the named transfer job with the given configuration.
func (j *Job) Update(ctx context.Context, name string, update *storagetransfer.UpdateTransferJobRequest) (*storagetransfer.TransferJob, error) {
	return j.service.TransferJobs.Patch(name, update).Context(ctx).Do()
}

// Operations lists all operations from the named job.
func (j *Job) Operations(ctx context.Context, name string, visit func(r *storagetransfer.ListOperationsResponse) error) error {
	f := filter{
		Project: j.project,
		Names:   []string{name},
	}
	bfilter, _ := json.Marshal(&f)
	return j.service.TransferOperations.List("transferOperations", string(bfilter)).Pages(ctx, visit)
}
