package stctl

import (
	"context"
	"errors"
	"testing"

	"github.com/m-lab/go/flagx"
	storagetransfer "google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Sync(t *testing.T) {
	tests := []struct {
		name    string
		c       *Command
		wantErr bool
	}{
		{
			name: "success",
			c: &Command{
				Job: &fakeTJ{
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							/*
								{
									Name:        "transferOperations/1234567890",
									Description: "description",
									Schedule: &storagetransfer.Schedule{
										ScheduleEndDate: nil,
										StartTimeOfDay: &storagetransfer.TimeOfDay{
											Hours:   10,
											Minutes: 9,
										},
									},
									TransferSpec: &storagetransfer.TransferSpec{
										GcsDataSource: &storagetransfer.GcsData{
											BucketName: "mlab-fake-source",
										},
										GcsDataSink: &storagetransfer.GcsData{
											BucketName: "mlab-fake-target",
										},
										ObjectConditions: &storagetransfer.ObjectConditions{
											IncludePrefixes: []string{"a", "b"},
										},
									},
								},
							*/
							{
								Name:        "transferOperations/9876543210",
								Description: "description2",
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: &storagetransfer.Date{
										Day:   1,
										Month: 2,
										Year:  2019,
									},
								},
							},
							{
								Name:        "transferOperations/1234567890-2",
								Description: getDesc("fake-source", "fake-target"),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay: &storagetransfer.TimeOfDay{
										Hours:   10,
										Minutes: 9,
									},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource: &storagetransfer.GcsData{
										BucketName: "fake-source",
									},
									GcsDataSink: &storagetransfer.GcsData{
										BucketName: "fake-target",
									},
									ObjectConditions: &storagetransfer.ObjectConditions{
										IncludePrefixes: []string{"a", "b"},
									},
								},
							},
						},
					},
					job: &storagetransfer.TransferJob{
						Description: getDesc("fake-source", "fake-target"),
					},
					updateErr: errors.New("fake disable error"),
				},
				SourceBucket: "fake-source",
				TargetBucket: "fake-target",
				Prefixes:     []string{"a", "b"},
				StartTime:    flagx.Time{Hour: 1, Minute: 2, Second: 3},
			},
			wantErr: true,
		},
		/*
			{
				name: "error-disable",
				c: &Command{
					Job: &fakeTJ{
						updateErr: errors.New("fake disable error"),
					},
				},
			},
		*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := tt.c.Sync(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Command.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
