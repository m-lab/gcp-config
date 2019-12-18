package stctl

import (
	"context"
	"fmt"
	"time"

	"github.com/m-lab/go/logx"
	"github.com/m-lab/go/pretty"

	"google.golang.org/api/storagetransfer/v1"
)

// Create creates a new storage transfer job.
func (c *Command) Create(ctx context.Context) (*storagetransfer.TransferJob, error) {
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
				Day:   int64(time.Now().UTC().Day()),
				Month: int64(time.Now().UTC().Month()),
				Year:  int64(time.Now().UTC().Year()),
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
	// On success, the returned job will include an assigned job.Name.
	return c.Client.Create(ctx, create)
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
