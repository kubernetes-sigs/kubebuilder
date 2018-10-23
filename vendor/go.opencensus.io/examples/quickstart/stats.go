// Copyright 2018, OpenCensus Authors
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

// Command stats implements the stats Quick Start example from:
//   https://opencensus.io/quickstart/go/metrics/
package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"net/http"

	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
	"go.opencensus.io/zpages"
)

var (
	// The latency in milliseconds
	MLatencyMs = stats.Float64("repl/latency", "The latency in milliseconds per REPL loop", "ms")

	// Counts the number of lines read in from standard input
	MLinesIn = stats.Int64("repl/lines_in", "The number of lines read in", "1")

	// Encounters the number of non EOF(end-of-file) errors.
	MErrors = stats.Int64("repl/errors", "The number of errors encountered", "1")

	// Counts/groups the lengths of lines read in.
	MLineLengths = stats.Int64("repl/line_lengths", "The distribution of line lengths", "By")
)

var (
	KeyMethod, _ = tag.NewKey("method")
)

var (
	LatencyView = &view.View{
		Name:        "demo/latency",
		Measure:     MLatencyMs,
		Description: "The distribution of the latencies",

		// Latency in buckets:
		// [>=0ms, >=25ms, >=50ms, >=75ms, >=100ms, >=200ms, >=400ms, >=600ms, >=800ms, >=1s, >=2s, >=4s, >=6s]
		Aggregation: view.Distribution(0, 25, 50, 75, 100, 200, 400, 600, 800, 1000, 2000, 4000, 6000),
		TagKeys:     []tag.Key{KeyMethod}}

	LineCountView = &view.View{
		Name:        "demo/lines_in",
		Measure:     MLinesIn,
		Description: "The number of lines from standard input",
		Aggregation: view.Count(),
	}

	ErrorCountView = &view.View{
		Name:        "demo/errors",
		Measure:     MErrors,
		Description: "The number of errors encountered",
		Aggregation: view.Count(),
	}

	LineLengthView = &view.View{
		Name:        "demo/line_lengths",
		Description: "Groups the lengths of keys in buckets",
		Measure:     MLineLengths,
		// Lengths: [>=0B, >=5B, >=10B, >=15B, >=20B, >=40B, >=60B, >=80, >=100B, >=200B, >=400, >=600, >=800, >=1000]
		Aggregation: view.Distribution(0, 5, 10, 15, 20, 40, 60, 80, 100, 200, 400, 600, 800, 1000),
	}
)

func main() {
	zpages.Handle(nil, "/debug")
	go http.ListenAndServe("localhost:8080", nil)

	// Create that Stackdriver stats exporter
	exporter, err := prometheus.NewExporter(prometheus.Options{})
	if err != nil {
		log.Fatalf("Failed to create the Stackdriver stats exporter: %v", err)
	}
	http.Handle("/metrics", exporter)

	// Register the stats exporter
	view.RegisterExporter(exporter)

	// Register the views
	if err := view.Register(LatencyView, LineCountView, ErrorCountView, LineLengthView); err != nil {
		log.Fatalf("Failed to register views: %v", err)
	}

	// But also we can change the metrics reporting period to 2 seconds
	//view.SetReportingPeriod(2 * time.Second)

	// In a REPL:
	//   1. Read input
	//   2. process input
	br := bufio.NewReader(os.Stdin)

	// repl is the read, evaluate, print, loop
	for {
		if err := readEvaluateProcess(br); err != nil {
			if err == io.EOF {
				return
			}
			log.Fatal(err)
		}
	}
}

// readEvaluateProcess reads a line from the input reader and
// then processes it. It returns an error if any was encountered.
func readEvaluateProcess(br *bufio.Reader) error {
	ctx, err := tag.New(context.Background(), tag.Insert(KeyMethod, "repl"))
	if err != nil {
		return err
	}

	fmt.Printf("> ")
	line, _, err := br.ReadLine()
	if err != nil {
		if err != io.EOF {
			stats.Record(ctx, MErrors.M(1))
		}
		return err
	}

	out, err := processLine(ctx, line)
	if err != nil {
		stats.Record(ctx, MErrors.M(1))
		return err
	}
	fmt.Printf("< %s\n\n", out)
	return nil
}

// processLine takes in a line of text and
// transforms it. Currently it just capitalizes it.
func processLine(ctx context.Context, in []byte) (out []byte, err error) {
	startTime := time.Now()
	defer func() {
		ms := float64(time.Since(startTime).Nanoseconds()) / 1e6
		stats.Record(ctx, MLinesIn.M(1), MLatencyMs.M(ms), MLineLengths.M(int64(len(in))))
	}()

	return bytes.ToUpper(in), nil
}
