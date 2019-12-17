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
	"fmt"
	"sort"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/logx"
	"github.com/m-lab/go/rtx"
	"github.com/stephen-soltesz/pretty"

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
	// NOTE: project_id is a required filter. (o_O)
	return c.Job.List(ctx, func(resp *storagetransfer.ListTransferJobsResponse) error {
		for _, job := range resp.TransferJobs {
			if job.Schedule.ScheduleEndDate == nil {
				logx.Debug.Print(pretty.Sprint(job))
				including := "All"
				if job.TransferSpec.ObjectConditions != nil {
					including = fmt.Sprintf("%v", job.TransferSpec.ObjectConditions.IncludePrefixes)
				}
				fmt.Printf("%-25s starting:%s desc:%q including:%v\n",
					job.Name, fmtTime(job.Schedule.StartTimeOfDay), job.Description, including)
			}
		}
		return nil
	})
}

// ListOperations lists past operations for the named job that started after c.AfterDate.
func (c *Command) ListOperations(ctx context.Context, name string) error {
	return c.Job.Operations(ctx, name, func(r *storagetransfer.ListOperationsResponse) error {
		for _, op := range r.Operations {
			m := parseJobMetadata(op.Metadata)
			if m.Start.Before(c.AfterDate) {
				continue
			}
			logx.Debug.Print(pretty.Sprint(op))
			if m.TransferSpec == nil {
				continue
			}
			fmt.Printf(
				("Copy %s to %s including:%v : " +
					"Found %-7s Copied %-7s Skipped %-8s Failed %2q :: " +
					"lasted %f minutes with Status %s and started at %s\n"),
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
	})
}

func fmtTime(t *storagetransfer.TimeOfDay) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds)
}

// Verify that the two times are equal.
func timesEqual(scheduled *storagetransfer.TimeOfDay, desired flagx.Time) bool {
	return fmtTime(scheduled) == desired.String()
}

// Verify that the two arrays have the same values.
func includesEqual(configured []string, desired []string) bool {
	if len(configured) != len(desired) {
		return false
	}
	sort.Strings(configured)
	sort.Strings(desired)
	for i := 0; i < len(configured); i++ {
		if configured[i] != desired[i] {
			return false
		}
	}
	return true
}

func specMatches(job *storagetransfer.TransferJob, start flagx.Time, prefixes []string) bool {
	if job.Schedule.StartTimeOfDay != nil &&
		!timesEqual(job.Schedule.StartTimeOfDay, start) {
		return false
	}
	if job.TransferSpec.ObjectConditions != nil &&
		!includesEqual(job.TransferSpec.ObjectConditions.IncludePrefixes, prefixes) {
		return false
	}
	return true
}

func parseJobMetadata(m googleapi.RawMessage) *jobMetadata {
	b, err := m.MarshalJSON()
	rtx.Must(err, "failed to marshal json of raw message")
	j := &jobMetadata{}
	rtx.Must(json.Unmarshal(b, j), "Failed to unmarshal jobmessage")
	return j
}

// getDesc returns the canonical description used to identify previously created
// jobs. WARNING: Do not modify this format without adjusting existing configs to match.
func getDesc(src, dest string) string {
	return fmt.Sprintf("STCTL: daily copy of %s to %s", src, dest)
}

func getSpec(src, dest string, prefixes []string) *storagetransfer.TransferSpec {
	spec := &storagetransfer.TransferSpec{
		GcsDataSource: &storagetransfer.GcsData{
			BucketName: src,
		},
		GcsDataSink: &storagetransfer.GcsData{
			BucketName: dest,
		},
	}
	if prefixes != nil {
		spec.ObjectConditions = &storagetransfer.ObjectConditions{
			IncludePrefixes: prefixes,
		}
	}
	return spec
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
