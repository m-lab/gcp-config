package stctl

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/pretty"
	"google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Create(t *testing.T) {
	expected := &storagetransfer.TransferJob{
		Description: "STCTL: daily copy of src-bucket to dest-bucket",
		Name:        "THIS-IS-A-FAKE-JOB-NAME",
		ProjectId:   "fake-mlab-testing",
		Schedule: &storagetransfer.Schedule{
			ScheduleEndDate: (*storagetransfer.Date)(nil),
			ScheduleStartDate: &storagetransfer.Date{
				Day:   18,
				Month: 12,
				Year:  2019,
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
			if !reflect.DeepEqual(job, expected) && !tt.wantErr {
				t.Errorf("Command.Create() different job returned; got %s\n, want %s\n", pretty.Sprint(job), pretty.Sprint(expected))
			}
		})
	}
}
