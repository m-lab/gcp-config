package stctl_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/m-lab/gcp-config/internal/stctl"
	"github.com/m-lab/go/flagx"
	storagetransfer "google.golang.org/api/storagetransfer/v1"
)

var getDesc = stctl.GetDesc

func TestCommand_Sync(t *testing.T) {
	ts := time.Now().UTC()
	tests := []struct {
		name        string
		c           *Command
		expected    *storagetransfer.TransferJob
		wantErr     bool
		shouldFind  bool // The specified job should be found in the client
		shouldMatch bool // The specified job should match the existing job.
	}{
		{
			name: "success-job-found-specs-match",
			c: &Command{
				Client: &fakeTJ{
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/ignore-job-with-end-date",
								Description: "ignore-job-with-end-date",
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: &storagetransfer.Date{Day: 1, Month: 2, Year: 2019},
								},
							},
							{
								Name:        "transferOperations/description-matches-gcs-buckets",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 1, Minute: 2, Second: 3}),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay:  &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
									GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
									ObjectConditions: &storagetransfer.ObjectConditions{
										IncludePrefixes:                     []string{"a", "b"},
										MaxTimeElapsedSinceLastModification: "432000s",
										MinTimeElapsedSinceLastModification: "3600s",
									},
									TransferOptions: &storagetransfer.TransferOptions{
										DeleteObjectsFromSourceAfterTransfer: true,
									},
								},
							},
						},
					},
				},
				SourceBucket:        "fake-source",
				TargetBucket:        "fake-target",
				Prefixes:            []string{"a", "b"},
				StartTime:           flagx.Time{Hour: 1, Minute: 2, Second: 3},
				MaxFileAge:          5 * 24 * time.Hour,
				MinFileAge:          time.Hour,
				DeleteAfterTransfer: true,
			},
			shouldFind:  true,
			shouldMatch: true,
			expected: &storagetransfer.TransferJob{
				Description: "STCTL: transfer fake-source -> fake-target at 01:02:03",
				Name:        "transferOperations/description-matches-gcs-buckets",
				Schedule: &storagetransfer.Schedule{
					StartTimeOfDay: &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
				},
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
					GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
					ObjectConditions: &storagetransfer.ObjectConditions{
						IncludePrefixes:                     []string{"a", "b"},
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
			name: "success-disable-and-create",
			c: &Command{
				SourceBucket: "fake-source",
				TargetBucket: "fake-target",
				Prefixes:     []string{"a", "b"},
				StartTime:    flagx.Time{Hour: 1, Minute: 2, Second: 3},
				Client: &fakeTJ{
					// fake jobs that are listed to search for one that matches the current Command spec.
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/description-matches-ObjectConditions-does-not",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 1, Minute: 2, Second: 3}),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay:  &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource:    &storagetransfer.GcsData{BucketName: "fake-source"},
									GcsDataSink:      &storagetransfer.GcsData{BucketName: "fake-target"},
									ObjectConditions: &storagetransfer.ObjectConditions{}, // Empty object conditions specified.
								},
							},
						},
					},
					// a fake job that is disabled.
					job: &storagetransfer.TransferJob{},
				},
			},
			shouldFind: true,
			expected: &storagetransfer.TransferJob{
				Description: "STCTL: transfer fake-source -> fake-target at 01:02:03",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				Schedule: &storagetransfer.Schedule{
					StartTimeOfDay: &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
				},
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
					GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
					ObjectConditions: &storagetransfer.ObjectConditions{
						IncludePrefixes: []string{"a", "b"},
					},
				},
				Status: "ENABLED",
			},
		},
		{
			name: "success-delete-mismatch-disable-and-create",
			c: &Command{
				SourceBucket:        "fake-source",
				TargetBucket:        "fake-target",
				StartTime:           flagx.Time{Hour: 1, Minute: 2, Second: 3},
				DeleteAfterTransfer: true,
				Client: &fakeTJ{
					// fake jobs that are listed to search for one that matches the current Command spec.
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/description-matches-delete-option-does-not",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 1, Minute: 2, Second: 3}),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay:  &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
									GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
								},
							},
						},
					},
					// a fake job that is disabled.
					job: &storagetransfer.TransferJob{},
				},
			},
			shouldFind: true,
			expected: &storagetransfer.TransferJob{
				Description: "STCTL: transfer fake-source -> fake-target at 01:02:03",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				Schedule: &storagetransfer.Schedule{
					StartTimeOfDay: &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
				},
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
					GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
					TransferOptions: &storagetransfer.TransferOptions{
						DeleteObjectsFromSourceAfterTransfer: true,
					},
				},
				Status: "ENABLED",
			},
		},
		{
			name: "success-nil-object-cond-disable-and-create",
			c: &Command{
				SourceBucket: "fake-source",
				TargetBucket: "fake-target",
				Prefixes:     []string{"a", "b"},
				StartTime:    flagx.Time{Hour: 1, Minute: 2, Second: 3},
				Client: &fakeTJ{
					// fake jobs that are listed to search for one that matches the current Command spec.
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/nil-object-cond",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 1, Minute: 2, Second: 3}),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay:  &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
									GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
								},
							},
						},
					},
					// a fake job that is disabled.
					job: &storagetransfer.TransferJob{},
				},
			},
			shouldFind: true,
			expected: &storagetransfer.TransferJob{
				Description: "STCTL: transfer fake-source -> fake-target at 01:02:03",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				Schedule: &storagetransfer.Schedule{
					StartTimeOfDay: &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
				},
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{BucketName: "fake-source"},
					GcsDataSink:   &storagetransfer.GcsData{BucketName: "fake-target"},
					ObjectConditions: &storagetransfer.ObjectConditions{
						IncludePrefixes: []string{"a", "b"},
					},
				},
				Status: "ENABLED",
			},
		},
		{
			name: "success-job-not-found-then-created",
			c: &Command{
				SourceBucket: "source",
				TargetBucket: "target",
				StartTime:    flagx.Time{Hour: 1, Minute: 2, Second: 3},
				Client: &fakeTJ{
					listJobResp: &storagetransfer.ListTransferJobsResponse{},
				},
			},
			expected: &storagetransfer.TransferJob{
				Description: "STCTL: transfer source -> target at 01:02:03",
				Name:        "THIS-IS-A-FAKE-ASSIGNED-JOB-NAME",
				Schedule: &storagetransfer.Schedule{
					StartTimeOfDay: &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
					ScheduleStartDate: &storagetransfer.Date{
						Day:   int64(ts.Day()),
						Month: int64(ts.Month()),
						Year:  int64(ts.Year()),
					},
				},
				TransferSpec: &storagetransfer.TransferSpec{
					GcsDataSource: &storagetransfer.GcsData{BucketName: "source"},
					GcsDataSink:   &storagetransfer.GcsData{BucketName: "target"},
				},
				Status: "ENABLED",
			},
		},
		{
			name: "error-list-jobs",
			c: &Command{
				Client: &fakeTJ{
					listErr: errors.New("Fake list error"),
				},
			},
			wantErr: true,
		},
		{
			name: "error-found-and-disable-error-different-IncludePrefixes",
			c: &Command{
				SourceBucket: "fake-source",
				TargetBucket: "fake-target",
				Prefixes:     []string{"a", "b"},
				StartTime:    flagx.Time{Hour: 1, Minute: 2, Second: 3},
				Client: &fakeTJ{
					// fake jobs that are listed to search for one that matches the current Command spec.
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/description-matches",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 1, Minute: 2, Second: 3}),
								Schedule: &storagetransfer.Schedule{
									ScheduleEndDate: nil,
									StartTimeOfDay:  &storagetransfer.TimeOfDay{Hours: 1, Minutes: 2, Seconds: 3},
								},
								TransferSpec: &storagetransfer.TransferSpec{
									GcsDataSource:    &storagetransfer.GcsData{BucketName: "fake-source"},
									GcsDataSink:      &storagetransfer.GcsData{BucketName: "fake-target"},
									ObjectConditions: &storagetransfer.ObjectConditions{IncludePrefixes: []string{"c", "d"}}, // IncludePrefixes do not match command.Prefixes.
								},
							},
						},
					},
					getErr: errors.New("fake get error causes Disable() to fail"),
				},
			},
			shouldFind: true,
			wantErr:    true,
		},
		{
			name: "error-found-and-disable-error-different-start-times",
			c: &Command{
				SourceBucket: "fake-source",
				TargetBucket: "fake-target",
				StartTime:    flagx.Time{Hour: 3, Minute: 2, Second: 1},
				Client: &fakeTJ{
					// fake jobs that are listed to search for one that matches the current Command spec.
					listJobResp: &storagetransfer.ListTransferJobsResponse{
						TransferJobs: []*storagetransfer.TransferJob{
							{
								Name:        "transferOperations/description-matches",
								Description: getDesc("fake-source", "fake-target", flagx.Time{Hour: 3, Minute: 2, Second: 1}),
								// Schedule can be empty because there is no TransferSpec?
								Schedule: &storagetransfer.Schedule{},
							},
						},
					},
					getErr: errors.New("fake get error causes Disable() to fail"),
				},
			},
			// With true wantErr, Schedule can be empty, and TransferSpec is not needed.
			// This does not impact test coverage.
			wantErr:    true,
			shouldFind: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			found, err := stctl.Find(tt.c, ctx)
			if !tt.wantErr && err != stctl.ErrNotFound && err != nil {
				t.Errorf("Command.Sync() error = %v", err)
			}
			if (found != nil) != tt.shouldFind {
				t.Errorf("Command.Sync() found = %v, shouldFind = %v", found, tt.shouldFind)
			}
			if found != nil {
				matches := stctl.SpecMatches(tt.c, found)
				if matches != tt.shouldMatch {
					t.Errorf("Command.Sync() matches = %v, shouldMatch = %v", matches, tt.shouldMatch)
				}
			}

			job, err := tt.c.Sync(ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Command.Sync() error = %v, wantErr %v", err, tt.wantErr)
			}
			// This only runs when !wantErr.  Otherwise, the fake job is never referenced.
			if diff := deep.Equal(job, tt.expected); diff != nil && !tt.wantErr {
				t.Errorf("Command.Sync() job did not match expected;\n%s", strings.Join(diff, "\n"))
			}
		})
	}
}
