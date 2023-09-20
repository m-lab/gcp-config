package stctl

import (
	"context"
	"fmt"
	"time"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/logx"
	"github.com/stephen-soltesz/pretty"

	"sort"

	"google.golang.org/api/storagetransfer/v1"
)

var (
	errNotFound = fmt.Errorf("no matching job found")
)

// find searches the Client for a TransferJob with matching Description.
// The returned TransferJob may or may not match the rest of the job spec.
func (c *Command) find(ctx context.Context) (*storagetransfer.TransferJob, error) {
	var found *storagetransfer.TransferJob

	// Generate canonical description from current config.
	desc := getDesc(c.SourceBucket, c.TargetBucket, c.StartTime)

	// List jobs and find first that matches canonical description.
	logx.Debug.Println("Listing jobs")
	findJob := func(resp *storagetransfer.ListTransferJobsResponse) error {
		for _, job := range resp.TransferJobs {
			if job.Schedule == nil || job.Schedule.ScheduleEndDate != nil {
				// We only manage jobs without an end date.
				continue
			}
			logx.Debug.Print(pretty.Sprint(job))
			if desc == job.Description {
				// Sync depends on the convention for storage transfer job management that
				// each job has a unique description, so the first
				// matching job should be the only matching job.
				found = job
				return nil
			}
		}
		// Job was not found.
		return errNotFound
	}

	err := c.Client.Jobs(ctx, findJob)
	return found, err
}

// Sync guarantees that a job exists matching the current command parameters. If
// a job with matching command parameters already exists, no action is taken. If
// a matching description is found with different values for IncludePrefixes or
// StartTimeOfDay, then the original job is disabled and a new job created. In
// either case, the found or newly created job is returned on success.
func (c *Command) Sync(ctx context.Context) (*storagetransfer.TransferJob, error) {
	found, err := c.find(ctx)
	if err != errNotFound && err != nil {
		return nil, err
	}
	if found != nil {
		logx.Debug.Println("Found job!")
		logx.Debug.Print(pretty.Sprint(found))
		if c.specMatches(found) {
			// We found a matching job, do nothing, return success.
			logx.Debug.Println("Specs match!")
			return found, nil
		}
		// We found a managed job and it does not match the new spec, so disable it.
		_, err := c.Disable(ctx, found.Name)
		if err != nil {
			return nil, err
		}
	}
	// Create new job matching the preferred spec.
	logx.Debug.Println("Creating new job!")
	return c.Create(ctx)
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

func (c *Command) specMatches(job *storagetransfer.TransferJob) bool {
	if job.Schedule.StartTimeOfDay == nil ||
		!timesEqual(job.Schedule.StartTimeOfDay, c.StartTime) {
		logx.Debug.Println("spec: times not equal", job.Schedule, c.StartTime)
		return false
	}

	if job.Schedule.RepeatInterval != c.Interval {
		logx.Debug.Println("spec: interval not equal", job.Schedule.RepeatInterval, c.Interval)
		return false
	}

	cond := job.TransferSpec.ObjectConditions
	if cond == nil {
		if len(c.Prefixes) > 0 || c.MaxFileAge > 0 || c.MinFileAge > 0 {
			logx.Debug.Println("spec: conditions not equal", cond, c.Prefixes, c.MaxFileAge, c.MinFileAge)
			return false
		}
	} else if !includesEqual(cond.IncludePrefixes, c.Prefixes) ||
		!durationsMatch(c.MaxFileAge, cond.MaxTimeElapsedSinceLastModification) ||
		!durationsMatch(c.MinFileAge, cond.MinTimeElapsedSinceLastModification) {
		logx.Debug.Println("spec: conditions not equal",
			cond.IncludePrefixes, c.Prefixes,
			cond.MaxTimeElapsedSinceLastModification, c.MaxFileAge.String(),
			cond.MinTimeElapsedSinceLastModification, c.MinFileAge.String())
		return false
	}

	jobDeleteOption := job.TransferSpec.TransferOptions != nil && job.TransferSpec.TransferOptions.DeleteObjectsFromSourceAfterTransfer
	if c.DeleteAfterTransfer != jobDeleteOption {
		logx.Debug.Println("spec: delete after transfer not equal", jobDeleteOption, c.DeleteAfterTransfer)
		return false
	}

	return true
}

// Convert the string based times from the ST API to numbers to make comparisons trivial.
func durationsMatch(age time.Duration, elapsed string) bool {
	// Accept that an error parsing correctly means zero seconds.
	d, _ := time.ParseDuration(elapsed)
	return age == d
}
