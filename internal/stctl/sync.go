package stctl

import (
	"context"
	"fmt"

	"github.com/m-lab/go/logx"
	"github.com/stephen-soltesz/pretty"

	"google.golang.org/api/storagetransfer/v1"
)

// Sync guarantees that a job exists matching the current command parameters.
// If a job with matching command parameters already exists, no action is taken.
// If a matching description is found with different values for IncludePrefixes
// or StartTimeOfDay, then the original job is disabled and a new job created.
func (c *Command) Sync(ctx context.Context) error {
	var found *storagetransfer.TransferJob
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
				found = job
				return errFound
			}
		}
		// Job was not found.
		return nil
	})
	if err != errFound && err != nil {
		return err
	}
	if found != nil {
		logx.Debug.Println("Found job!")
		logx.Debug.Print(pretty.Sprint(found))
		if specMatches(found, c.StartTime, c.Prefixes) {
			// We found a matching job, do nothing, return success.
			logx.Debug.Println("Specs match!")
			return nil
		}
		// We found a managed job and it does not match the new spec, so disable it.
		err = c.Disable(ctx, found.Name)
		if err != nil {
			return err
		}
	}
	// Create new job matching the preferred spec.
	return c.Create(ctx)
}
