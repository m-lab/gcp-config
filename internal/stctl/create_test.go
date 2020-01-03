package stctl

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
	expected := &storagetransfer.TransferJob{
		Description: "STCTL: daily copy of src-bucket to dest-bucket",
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
				IncludePrefixes: []string{"ndt"},
			},
		},
	}

	tests := []struct {
		name    string
		c       *Command
		wantErr bool
	}{
		{
			name: "success",
			c: &Command{
				Client:       &fakeTJ{},
				Prefixes:     []string{"ndt"},
				StartTime:    flagx.Time{Hour: 2, Minute: 10},
				SourceBucket: "src-bucket",
				TargetBucket: "dest-bucket",
				Project:      "fake-mlab-testing",
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
				t.Errorf("Command.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := deep.Equal(job, expected); diff != nil && !tt.wantErr {
				t.Errorf("Command.Create() returned and expected jobs differ; %v", diff)
			}
		})
	}
}