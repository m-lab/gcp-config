package stctl_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/m-lab/go/flagx"
	"google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Create(t *testing.T) {
	ts := time.Now().UTC()
	tests := []struct {
		name         string
		c            *Command
		expect       storagetransfer.TransferJob
		shouldDelete bool
		wantErr      bool
	}{
		{
			name: "success-no-delete",
			c: &Command{
				Client:              &fakeTJ{},
				Prefixes:            []string{"ndt"},
				StartTime:           flagx.Time{Hour: 2, Minute: 10},
				SourceBucket:        "src-bucket",
				TargetBucket:        "dest-bucket",
				MaxFileAge:          5 * 24 * time.Hour,
				MinFileAge:          time.Hour,
				Project:             "fake-mlab-testing",
				DeleteAfterTransfer: false,
			},
			expect: storagetransfer.TransferJob{
				Description: "STCTL: transfer src-bucket -> dest-bucket at 02:10:00",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				ProjectId:   "fake-mlab-testing",
				Schedule: &storagetransfer.Schedule{
					ScheduleEndDate: (*storagetransfer.Date)(nil),
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
					StartTimeOfDay: &storagetransfer.TimeOfDay{
						Hours:   2,
						Minutes: 10,
					},
				},
				Status: "ENABLED",
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{
						BucketName: "src-bucket",
					},
					GcsDataSink: &storagetransfer.GcsData{
						BucketName: "dest-bucket",
					},
					ObjectConditions: &storagetransfer.ObjectConditions{
						IncludePrefixes:                     []string{"ndt"},
						MaxTimeElapsedSinceLastModification: "432000s",
						MinTimeElapsedSinceLastModification: "3600s",
					},
				},
			},
		},

		{
			name: "success-delete-after-transfer",
			c: &Command{
				Client:              &fakeTJ{},
				Prefixes:            []string{"ndt"},
				StartTime:           flagx.Time{Hour: 2, Minute: 10},
				SourceBucket:        "src-bucket",
				TargetBucket:        "dest-bucket",
				MaxFileAge:          5 * 24 * time.Hour,
				MinFileAge:          time.Hour,
				Project:             "fake-mlab-testing",
				DeleteAfterTransfer: true,
			},
			expect: storagetransfer.TransferJob{
				Description: "STCTL: transfer src-bucket -> dest-bucket at 02:10:00",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				ProjectId:   "fake-mlab-testing",
				Schedule: &storagetransfer.Schedule{
					ScheduleEndDate: (*storagetransfer.Date)(nil),
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
					StartTimeOfDay: &storagetransfer.TimeOfDay{
						Hours:   2,
						Minutes: 10,
					},
				},
				Status: "ENABLED",
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{
						BucketName: "src-bucket",
					},
					GcsDataSink: &storagetransfer.GcsData{
						BucketName: "dest-bucket",
					},
					ObjectConditions: &storagetransfer.ObjectConditions{
						IncludePrefixes:                     []string{"ndt"},
						MaxTimeElapsedSinceLastModification: "432000s",
						MinTimeElapsedSinceLastModification: "3600s",
					},
					TransferOptions: &storagetransfer.TransferOptions{
						DeleteObjectsFromSourceAfterTransfer: true,
					},
				},
			},
		},
		{
			name: "error-create",
			c: &Command{
				Client: &fakeTJ{
					createErr: errors.New("Fake create error"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			job, err := tt.c.Create(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Command.Create(%s) error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
			if diff := deep.Equal(job, &tt.expect); diff != nil && !tt.wantErr {
				t.Errorf("Command.Create(%s) returned and expected jobs differ; %v", tt.name, diff)
			}
		})
	}
}
