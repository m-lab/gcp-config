package stctl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/m-lab/go/flagx"
)

func TestCommand_Create(t *testing.T) {
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
		wantErr bool
	}{
		{
			name: "success",
			fields: fields{
				Job: &fakeTJ{},
			},
		},
		{
			name: "error-create",
			fields: fields{
				Job: &fakeTJ{
					createErr: errors.New("Fake create error"),
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
			if err := c.Create(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Command.Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
