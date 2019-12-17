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

// Package stctl implements command actions.
package stctl

import (
	"context"
	"encoding/json"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/rtx"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storagetransfer/v1"
)

// TransferJob captures the interface required by Command implementations.
type TransferJob interface {
	List(ctx context.Context, visit func(resp *storagetransfer.ListTransferJobsResponse) error) error
	Create(ctx context.Context, create *storagetransfer.TransferJob) (*storagetransfer.TransferJob, error)
	Get(ctx context.Context, name string) (*storagetransfer.TransferJob, error)
	Update(ctx context.Context, name string, update *storagetransfer.UpdateTransferJobRequest) (*storagetransfer.TransferJob, error)
	Operations(ctx context.Context, name string, visit func(r *storagetransfer.ListOperationsResponse) error) error
}

// Command executes stctl actions.
type Command struct {
	Job          TransferJob
	Project      string
	SourceBucket string
	TargetBucket string
	Prefixes     []string
	StartTime    flagx.Time
	AfterDate    time.Time
}

// ListJobs lists enabled transfer jobs.
func (c *Command) ListJobs(ctx context.Context) error {
	return nil
}

// ListOperations lists past operations for the named job that started after c.AfterDate.
func (c *Command) ListOperations(ctx context.Context, name string) error {
	return nil
}

// Create creates a new storage transfer job.
func (c *Command) Create(ctx context.Context) error {
	return nil
}

// Disable marks the job status as 'DISABLED'.
func (c *Command) Disable(ctx context.Context, name string) error {
	return nil
}

// Sync guarantees that a job exists matching the current command parameters.
// If a job with matching command parameters already exists, no action is taken.
// If a matching description is found with different values for IncludePrefixes
// or StartTimeOfDay, then the original job is disabled and a new job created.
func (c *Command) Sync(ctx context.Context) error {
	return nil
}

// The Metadata field of storagetransfer.TransferOperation must be parsed from a
// JSON blob. The structs below are a subset of fields available.
func parseJobMetadata(m googleapi.RawMessage) *jobMetadata {
	b, err := m.MarshalJSON()
	rtx.Must(err, "failed to marshal json of raw message")
	j := &jobMetadata{}
	rtx.Must(json.Unmarshal(b, j), "Failed to unmarshal jobmessage")
	return j
}

type counters struct {
	ObjectsFound                   string `json:"objectsFoundFromSource"`
	ObjectsCopied                  string `json:"objectsCopiedToSink"`
	ObjectsFromSourceSkippedBySync string `json:"objectsFromSourceSkippedBySync"`
	ObjectsFromSourceFailed        string `json:"objectsFromSourceFailed"`
}

type jobMetadata struct {
	TransferSpec *storagetransfer.TransferSpec `json:"transferSpec"`
	Start        time.Time                     `json:"startTime"`
	End          time.Time                     `json:"endTime"`
	Counters     counters                      `json:"counters"`
	Status       string                        `json:"status"`
}
