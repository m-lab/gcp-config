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

func fmtTime(t *storagetransfer.TimeOfDay) string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds)
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
				fmt.Printf("%-25s starting:%s desc:%q including:%v\n", job.Name, fmtTime(job.Schedule.StartTimeOfDay), job.Description, including)
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
			fmt.Printf("Copy %s to %s including:%v : Found %-7s Copied %-7s Skipped %-8s Failed %2q :: lasted %f minutes with Status %s and started at %s\n",
				m.TranserSpec.GcsDataSource.BucketName, m.TranserSpec.GcsDataSink.BucketName,
				m.TranserSpec.ObjectConditions.IncludePrefixes,
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

// Disable marks the job status as 'DISABLED'.
func (c *Command) Disable(ctx context.Context, name string) error {
	current, err := c.Job.Get(ctx, name)
	if err != nil {
		return err
	}
	// NOTE: "Updating a job's schedule is not allowed."
	// https://godoc.org/google.golang.org/api/storagetransfer/v1#TransferJobsService.Patch
	update := &storagetransfer.UpdateTransferJobRequest{
		ProjectId: c.Project,
		// MUST: only set three fields: `Description`, `TransferSpec`, and `Status`.
		TransferJob: &storagetransfer.TransferJob{
			Description:  current.Description,
			TransferSpec: current.TransferSpec,
			// "ENABLED", "DISABLED", "DELETED"
			// NOTE: we prefer disabled status to preserve transfer history in the web UI.
			Status: "DISABLED", // No longer scheduled. Still visible in web UI.
		},
	}
	logx.Debug.Print(pretty.Sprint(update))
	job, err := c.Job.Update(ctx, name, update)
	if err != nil {
		return err
	}
	pretty.Print(job)
	return nil
}

// Create creates a new storage transfer job.
func (c *Command) Create(ctx context.Context) error {
	spec := getSpec(c.SourceBucket, c.TargetBucket, c.Prefixes)
	desc := getDesc(c.SourceBucket, c.TargetBucket)
	create := &storagetransfer.TransferJob{
		Description: desc,
		ProjectId:   c.Project,
		Schedule: &storagetransfer.Schedule{
			// Our transfers will have no end date. We want them to run indefinitely.
			ScheduleEndDate: nil,
			// Date to start transfers. May start today if StartTimeOfDay is in the future.
			// If StartTimeOfDay is in the past, then the first transfer will be scheduled tomorrow.
			ScheduleStartDate: &storagetransfer.Date{
				Day:   int64(time.Now().Day()),
				Month: int64(time.Now().Month()),
				Year:  int64(time.Now().Year()),
			},
			StartTimeOfDay: &storagetransfer.TimeOfDay{
				Hours:   int64(c.StartTime.Hour),
				Minutes: int64(c.StartTime.Minute),
				Seconds: int64(c.StartTime.Second),
			},
		},
		Status:       "ENABLED",
		TransferSpec: spec,
	}
	logx.Debug.Print(pretty.Sprint(create))
	job, err := c.Job.Create(ctx, create)
	if err != nil {
		return err
	}
	pretty.Print(job)
	return nil
}

// Verify that the two times are equal.
func timesEqual(scheduled *storagetransfer.TimeOfDay, desired flagx.Time) bool {
	return scheduled.Hours == int64(desired.Hour) &&
		scheduled.Minutes == int64(desired.Minute) &&
		scheduled.Seconds == int64(desired.Second)
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

func specMatches(foundJob *storagetransfer.TransferJob, start flagx.Time, prefixes []string) bool {
	if foundJob.Schedule.StartTimeOfDay != nil &&
		!timesEqual(foundJob.Schedule.StartTimeOfDay, start) {
		return false
	}
	if foundJob.TransferSpec.ObjectConditions != nil &&
		!includesEqual(foundJob.TransferSpec.ObjectConditions.IncludePrefixes, prefixes) {
		return false
	}
	return true
}

func datesEqual(start *storagetransfer.Date, end *storagetransfer.Date) bool {
	return (start != nil && end != nil && start.Day == end.Day && start.Month == end.Month && start.Year == end.Year)
}

// Sync guarantees that a job exists matching the current parameters for Create.
// However, unlike Create, if a similar job already exists, no action is taken.
// If a matching description is found with different values for IncludePrefixes
// or StartTimeOfDay, then the original job is disabled and a new job created.
func (c *Command) Sync(ctx context.Context) error {
	var foundJob *storagetransfer.TransferJob
	errFound := fmt.Errorf("Found matching job")
	// Generate canonical description from current config.
	desc := getDesc(c.SourceBucket, c.TargetBucket)
	// List jobs and find first that matches canonical description.
	logx.Debug.Println("Listing jobs")
	err := c.Job.List(ctx, func(resp *storagetransfer.ListTransferJobsResponse) error {
		for _, job := range resp.TransferJobs {
			if job.Schedule.ScheduleEndDate != nil {
				// We only manage jobs without an end date.
				continue
			}
			logx.Debug.Print(pretty.Sprint(job))
			if desc == job.Description {
				// By convention there will only be a single transfer job between two buckets.
				foundJob = job
				return errFound
			}
		}
		// Job was not found.
		return nil
	})
	if err != errFound && err != nil {
		return err
	}
	if foundJob != nil {
		logx.Debug.Println("Found job:")
		logx.Debug.Print(pretty.Sprint(foundJob))
		if specMatches(foundJob, c.StartTime, c.Prefixes) {
			// We found a matching job, do nothing, return success.
			logx.Debug.Println("Specs match!:")
			return nil
		}
		// We found a managed job and it does not match the new spec, so disable it.
		err = c.Disable(ctx, foundJob.Name)
		if err != nil {
			return err
		}
	}
	// Create new job matching the preferred spec.
	return c.Create(ctx)
}

func parseJobMetadata(m googleapi.RawMessage) *jobMetadata {
	b, err := m.MarshalJSON()
	rtx.Must(err, "failed to marshal json of raw message")
	j := &jobMetadata{}
	rtx.Must(json.Unmarshal(b, j), "Failed to unmarshal jobmessage")
	return j
}

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
	TranserSpec *storagetransfer.TransferSpec `json:"transferSpec"`
	Start       time.Time                     `json:"startTime"`
	End         time.Time                     `json:"endTime"`
	Counters    counters                      `json:"counters"`
	Status      string                        `json:"status"`
}
