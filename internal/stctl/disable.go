package stctl

import (
	"context"

	"github.com/m-lab/go/logx"
	"github.com/stephen-soltesz/pretty"

	"google.golang.org/api/storagetransfer/v1"
)

// Disable marks the job status as 'DISABLED'.
func (c *Command) Disable(ctx context.Context, name string) (*storagetransfer.TransferJob, error) {
	current, err := c.Client.Get(ctx, name)
	if err != nil {
		return nil, err
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
	job, err := c.Client.Update(ctx, name, update)
	if err != nil {
		return nil, err
	}
	return job, nil
}
