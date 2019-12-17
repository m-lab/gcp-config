package stctl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/m-lab/go/flagx"
	storagetransfer "google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Disable(t *testing.T) {
	type fields struct {
		Job          TransferJob
		Project      string
		SourceBucket string
		TargetBucket string
		Prefixes     []string
		StartTime    flagx.Time
		AfterDate    time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		job     string
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Job: &fakeTJ{
					job: &storagetransfer.TransferJob{
						Name:        "name",
						Description: "This is a description",
						Status:      "OK",
					},
				},
			},
		},
		{
			name: "error-get",
			fields: fields{
				Job: &fakeTJ{
					getErr: errors.New("Fake error calling Job.Get"),
				},
			},
			wantErr: true,
		},
		{
			name: "error-update",
			fields: fields{
				Job: &fakeTJ{
					job: &storagetransfer.TransferJob{
						Name:        "name",
						Description: "This is a description",
						Status:      "OK",
					},
					updateErr: errors.New("Fake error calling Job.Update"),
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Command{
				Job:          tt.fields.Job,
				Project:      tt.fields.Project,
				SourceBucket: tt.fields.SourceBucket,
				TargetBucket: tt.fields.TargetBucket,
				Prefixes:     tt.fields.Prefixes,
				StartTime:    tt.fields.StartTime,
				AfterDate:    tt.fields.AfterDate,
			}
			ctx := context.Background()
			if err := c.Disable(ctx, tt.job); (err != nil) != tt.wantErr {
				t.Errorf("Command.Disable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
