// Copyright © 2019 gcp-config Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package stctl implements command actions.
package stctl

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/m-lab/go/rtx"
	"google.golang.org/api/storagetransfer/v1"
)

type fakeTJ struct {
	listJobResp *storagetransfer.ListTransferJobsResponse
	listOpsResp *storagetransfer.ListOperationsResponse
	job         *storagetransfer.TransferJob
	getErr      error
	updateErr   error
	createErr   error
}

func (f *fakeTJ) Jobs(ctx context.Context, visit func(resp *storagetransfer.ListTransferJobsResponse) error) error {
	return visit(f.listJobResp)
}

func (f *fakeTJ) Create(ctx context.Context, create *storagetransfer.TransferJob) (*storagetransfer.TransferJob, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	create.Name = "THIS-IS-A-FAKE-JOB-NAME"
	return create, nil
}

func (f *fakeTJ) Get(ctx context.Context, name string) (*storagetransfer.TransferJob, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.job, nil
}

func (f *fakeTJ) Update(ctx context.Context, name string, update *storagetransfer.UpdateTransferJobRequest) (*storagetransfer.TransferJob, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.job != nil {
		f.job.Status = update.TransferJob.Status
	}
	return f.job, nil
}

func (f *fakeTJ) Operations(ctx context.Context, name string, visit func(r *storagetransfer.ListOperationsResponse) error) error {
	return visit(f.listOpsResp)
}

func TestCommand_ListJobs(t *testing.T) {
	output := &bytes.Buffer{}
	tests := []struct {
		name    string
		c       *Command
		wantErr bool
	}{
		{
			name: "success",
			c: &Command{
				Output: output,
				Client: &fakeTJ{
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
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
									ObjectConditions: &storagetransfer.ObjectConditions{
										IncludePrefixes: []string{"a", "b"},
									},
								},
							},
						},
					},
				},
				Project: "fake-mlab-testing",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := tt.c.ListJobs(ctx); (err != nil) != tt.wantErr {
				t.Errorf("Command.ListJobs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	c := strings.Count(output.String(), "\n")
	if c != 1 {
		t.Errorf("Command.ListJobs() wrote wrong number of lines; got %d, want 1", c)
	}
}

func md2JSON(m jobMetadata) []byte {
	b, err := json.Marshal(m)
	rtx.Must(err, "Failed to marshal jobMetadata")
	return b
}

func TestCommand_ListOperations(t *testing.T) {
	output := &bytes.Buffer{}
	tests := []struct {
		name    string
		c       *Command
		job     string
		wantErr bool
	}{
		{
			name: "success",
			c: &Command{
				Output:  output,
				Project: "fake-mlab-testing",
				Client: &fakeTJ{
					listOpsResp: &storagetransfer.ListOperationsResponse{
						Operations: []*storagetransfer.Operation{
							{
								Name: "transferOperations/1234567890",
								Metadata: md2JSON(jobMetadata{
									TransferSpec: &storagetransfer.TransferSpec{
										GcsDataSource: &storagetransfer.GcsData{
											BucketName: "mlab-fake-source",
										},
										GcsDataSink: &storagetransfer.GcsData{
											BucketName: "mlab-fake-target",
										},
										ObjectConditions: &storagetransfer.ObjectConditions{
											IncludePrefixes: []string{"ndt"},
										},
									},
									Start: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC),
									End:   time.Date(2019, time.January, 1, 1, 0, 0, 0, time.UTC),
								}),
							},
						},
					},
				},
			},
		},
		{
			name: "skip-after-date",
			c: &Command{
				Output:    output,
				Project:   "fake-mlab-testing",
				AfterDate: time.Date(2019, time.January, 1, 0, 0, 0, 0, time.UTC),
				Client: &fakeTJ{
					listOpsResp: &storagetransfer.ListOperationsResponse{
						Operations: []*storagetransfer.Operation{
							{
								Name: "transferOperations/afterdate-excludes-operation-dates",
								Metadata: md2JSON(jobMetadata{
									Start: time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC),
									End:   time.Date(2018, time.January, 1, 1, 0, 0, 0, time.UTC),
								}),
							},
						},
					},
				},
			},
		},
		{
			name: "skip-missing-transferspec",
			c: &Command{
				Output: output,
				Client: &fakeTJ{
					listOpsResp: &storagetransfer.ListOperationsResponse{
						Operations: []*storagetransfer.Operation{
							{
								Name: "transferOperations/missing-transferspec",
								Metadata: md2JSON(jobMetadata{
									Start: time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC),
									End:   time.Date(2018, time.January, 1, 1, 0, 0, 0, time.UTC),
								}),
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if err := tt.c.ListOperations(ctx, tt.job); (err != nil) != tt.wantErr {
				t.Errorf("Command.ListOperations() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	c := strings.Count(output.String(), "\n")
	if c != 1 {
		t.Errorf("Command.ListOperations() wrote wrong number of lines; got %d, want 1", c)
	}
}
