package stctl

import (
	"context"
	"errors"
	"testing"

	"github.com/go-test/deep"
	storagetransfer "google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Disable(t *testing.T) {
	expected := &storagetransfer.TransferJob{
		Name:        "job-name",
		Description: "This is the job description",
		Status:      "DISABLED",
		TransferSpec: &storagetransfer.TransferSpec{
			GcsDataSource: &storagetransfer.GcsData{
				BucketName: "source",
			},
			GcsDataSink: &storagetransfer.GcsData{
				BucketName: "destination",
			},
		},
	}
	tests := []struct {
		name    string
		c       *Command
		job     string
		wantErr bool
	}{
		{
			name: "success",
			c: &Command{
				Client: &fakeTJ{
					job: &storagetransfer.TransferJob{
						Name:        "job-name",
						Description: "This is the job description",
						Status:      "NOT-DISABLED",
						TransferSpec: &storagetransfer.TransferSpec{
							GcsDataSource: &storagetransfer.GcsData{
								BucketName: "source",
							},
							GcsDataSink: &storagetransfer.GcsData{
								BucketName: "destination",
							},
						},
					},
				},
			},
		},
		{
			name: "error-get",
			c: &Command{
				Client: &fakeTJ{
					getErr: errors.New("Fake error calling Job.Get"),
				},
			},
			wantErr: true,
		},
		{
			name: "error-update",
			c: &Command{
				Client: &fakeTJ{
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
			ctx := context.Background()
			job, err := tt.c.Disable(ctx, tt.job)
			if (err != nil) != tt.wantErr {
				t.Errorf("Command.Disable() error = %v, wantErr %v", err, tt.wantErr)
			}
			if diff := deep.Equal(job, expected); diff != nil && !tt.wantErr {
				t.Errorf("Command.Disable() job did not match expected; %v", diff)
			}
		})
	}
}
