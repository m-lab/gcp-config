// Copyright © 2019 gcp-config Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/m-lab/gcp-config/internal/stctl"
	"github.com/m-lab/gcp-config/transfer"

	"github.com/m-lab/go/flagx"
	"github.com/m-lab/go/pretty"
	"github.com/m-lab/go/rtx"

	"google.golang.org/api/storagetransfer/v1"
)

var (
	project             string
	sourceBucket        string
	destBucket          string
	prefixes            flagx.StringArray
	startTime           flagx.Time
	interval            string
	afterDate           flagx.DateTime
	minAge              time.Duration
	maxAge              time.Duration
	deleteAfterTransfer bool
)

func init() {
	flag.StringVar(&project, "project-id", "", "GCP project to sync transfer job.")
	flag.StringVar(&sourceBucket, "gcs.source", "", "Source GCS bucket.")
	flag.StringVar(&destBucket, "gcs.target", "", "Destination bucket.")
	flag.Var(&prefixes, "include", "Only transfer files with given prefix. Default all prefixes. Can be specified multiple times.")
	flag.Var(&startTime, "time", "Start daily transfer at this time (HH:MM:SS)")
	flag.StringVar(&interval, "interval", "", "Interval between the start of each scheduled transfer operation.")
	flag.Var(&afterDate, "after", "Only list operations that ran after the given date. Default is all dates.")
	flag.DurationVar(&minAge, "minFileAge", 0, "Minimum time since file modification")
	flag.DurationVar(&maxAge, "maxFileAge", 0, "Maximum time since file modification")
	flag.BoolVar(&deleteAfterTransfer, "deleteAfterTransfer", false, "Whether to delete source files after transfer")
}

var usageText = `
NAME
  stctl - storage transfer control

DESCRIPTION
  stctl allows a user to create, disable, and list storage transfer jobs and
  list past transfer operations for existing jobs.

EXAMPLES
  stctl -project-id <project> list

  stctl -project-id <project> operations <job name>

  stctl -project-id <project> create -gcs.source <bucket> -gcs.target <bucket> \
	-time <HH:MM:SS> -maxAge <duration> -minAge <duration> -deleteAfterTransfer true \
    -include ndt -include host -include neubot -include utilization

  stctl -project-id <project> disable <job name>

USAGE
`

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), usageText)
		flag.PrintDefaults()
	}
}

func mustArg(n int) string {
	args := flag.Args()
	if len(args)-1 < n {
		flag.Usage()
		os.Exit(1)
	}
	return args[n]
}

func main() {
	flag.Parse()
	rtx.Must(flagx.ArgsFromEnv(flag.CommandLine), "Failed to parse flags")

	ctx := context.Background()
	service, err := storagetransfer.NewService(ctx)
	rtx.Must(err, "Failed to create new storage transfer service")

	cmd := &stctl.Command{
		Client:              transfer.NewJob(project, service),
		Project:             project,
		SourceBucket:        sourceBucket,
		TargetBucket:        destBucket,
		Prefixes:            prefixes,
		StartTime:           startTime,
		Interval:            interval,
		AfterDate:           afterDate.Time,
		MinFileAge:          minAge.Truncate(time.Second),
		MaxFileAge:          maxAge.Truncate(time.Second),
		DeleteAfterTransfer: deleteAfterTransfer,
		Output:              os.Stdout,
	}

	op := mustArg(0)
	switch op {
	case "create":
		job, err := cmd.Create(ctx)
		rtx.Must(err, "Failed to create")
		pretty.Print(job)
	case "sync":
		job, err := cmd.Sync(ctx)
		rtx.Must(err, "Failed to sync")
		pretty.Print(job)
	case "disable":
		name := mustArg(1)
		job, err := cmd.Disable(ctx, name)
		rtx.Must(err, "Failed to disable %q", name)
		pretty.Print(job)
	case "list":
		rtx.Must(cmd.ListJobs(ctx), "Failed to list jobs")
	case "operations":
		name := mustArg(1)
		rtx.Must(cmd.ListOperations(ctx, name), "Failed to list operations for %q", name)
	default:
		flag.Usage()
	}
}
