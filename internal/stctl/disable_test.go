package stctl

import (
	"context"
	"errors"
	"testing"

	storagetransfer "google.golang.org/api/storagetransfer/v1"
)

func TestCommand_Disable(t *testing.T) {
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
						Name:        "name",
						Description: "This is a description",
						Status:      "OK",
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
			if err := tt.c.Disable(ctx, tt.job); (err != nil) != tt.wantErr {
				t.Errorf("Command.Disable() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
