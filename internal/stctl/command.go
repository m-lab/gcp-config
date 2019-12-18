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

// Package stctl implements storage tranfer actions used by the stctl CLI tool.
package stctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/logx"
	"github.com/m-lab/go/pretty"
	"github.com/m-lab/go/rtx"

	"google.golang.org/api/googleapi"
	"google.golang.org/api/storagetransfer/v1"
)

// TransferJob captures the interface required by Command implementations.
type TransferJob interface {
	Jobs(ctx context.Context, visit func(resp *storagetransfer.ListTransferJobsResponse) error) error
	Create(ctx context.Context, create *storagetransfer.TransferJob) (*storagetransfer.TransferJob, error)
	Get(ctx context.Context, name string) (*storagetransfer.TransferJob, error)
	Update(ctx context.Context, name string, update *storagetransfer.UpdateTransferJobRequest) (*storagetransfer.TransferJob, error)
	Operations(ctx context.Context, name string, visit func(r *storagetransfer.ListOperationsResponse) error) error
}

// Command executes stctl actions.
type Command struct {
	Client       TransferJob
	Project      string
	SourceBucket string
	TargetBucket string
	Prefixes     []string
	StartTime    flagx.Time
	AfterDate    time.Time
	Output       io.Writer
}

// ListJobs lists enabled transfer jobs.
func (c *Command) ListJobs(ctx context.Context) error {
	visit := func(resp *storagetransfer.ListTransferJobsResponse) error {
		for _, job := range resp.TransferJobs {
			// NB: One-time jobs have equal ScheduleStartDate and ScheduleEndDate.
			// We only manage daily jobs that never terminate, which have no ScheduleEndDate.
			if job.Schedule.ScheduleEndDate == nil {
				logx.Debug.Print(pretty.Sprint(job))
				including := "AllPrefixes"
				if job.TransferSpec.ObjectConditions != nil {
					including = fmt.Sprintf("%v", job.TransferSpec.ObjectConditions.IncludePrefixes)
				}
				fmt.Fprintf(c.Output, "%-25s starting:%s desc:%q including:%v\n",
					job.Name, fmtTime(job.Schedule.StartTimeOfDay), job.Description, including)
			}
		}
		return nil
	}
	return c.Client.Jobs(ctx, visit)
}

// ListOperations lists past operations for the named job that started after c.AfterDate.
func (c *Command) ListOperations(ctx context.Context, name string) error {
	visit := func(r *storagetransfer.ListOperationsResponse) error {
		for _, op := range r.Operations {
			m := parseJobMetadata(op.Metadata)
			if m.Start.Before(c.AfterDate) {
				// Ignore operations before AfterDate.
				continue
			}
			logx.Debug.Print(pretty.Sprint(op))
			if m.TransferSpec == nil || m.TransferSpec.ObjectConditions == nil {
				continue
			}
			fmt.Fprintf(c.Output,
				("Copy %s to %s including:%v :: " +
					"Found %-7s Copied %-7s Skipped %-8s Failed %2q :: " +
					"Lasted %f minutes with Status %s Started %s\n"),
				m.TransferSpec.GcsDataSource.BucketName, m.TransferSpec.GcsDataSink.BucketName,
				m.TransferSpec.ObjectConditions.IncludePrefixes,
				m.Counters.ObjectsFound,
				m.Counters.ObjectsCopied,
				m.Counters.ObjectsFromSourceSkippedBySync,
				m.Counters.ObjectsFromSourceFailed,
				m.End.Sub(m.Start).Minutes(),
				m.Status,
				m.Start,
			)
		}
		return nil
	}
	return c.Client.Operations(ctx, name, visit)
}

func fmtTime(t *storagetransfer.TimeOfDay) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds)
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
