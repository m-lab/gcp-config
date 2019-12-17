package stctl

import (
	"context"
	"time"

	"github.com/m-lab/go/logx"
	"github.com/stephen-soltesz/pretty"

	"google.golang.org/api/storagetransfer/v1"
)

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
