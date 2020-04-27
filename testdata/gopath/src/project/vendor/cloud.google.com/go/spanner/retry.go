/*
Copyright 2017 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package spanner

import (
	"context"
	"time"

	"cloud.google.com/go/internal/trace"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/googleapis/gax-go/v2"
	edpb "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

const (
	retryInfoKey = "google.rpc.retryinfo-bin"
)

// DefaultRetryBackoff is used for retryers as a fallback value when the server
// did not return any retry information.
var DefaultRetryBackoff = gax.Backoff{
	Initial:    20 * time.Millisecond,
	Max:        32 * time.Second,
	Multiplier: 1.3,
}

// spannerRetryer extends the generic gax Retryer, but also checks for any
// retry info returned by Cloud Spanner and uses that if present.
type spannerRetryer struct {
	gax.Retryer
}

// onCodes returns a spannerRetryer that will retry on the specified error
// codes.
func onCodes(bo gax.Backoff, cc ...codes.Code) gax.Retryer {
	return &spannerRetryer{
		Retryer: gax.OnCodes(cc, bo),
	}
}

// Retry returns the retry delay returned by Cloud Spanner if that is present.
// Otherwise it returns the retry delay calculated by the generic gax Retryer.
func (r *spannerRetryer) Retry(err error) (time.Duration, bool) {
	delay, shouldRetry := r.Retryer.Retry(err)
	if !shouldRetry {
		return 0, false
	}
	if serverDelay, hasServerDelay := extractRetryDelay(err); hasServerDelay {
		delay = serverDelay
	}
	return delay, true
}

// runWithRetryOnAborted executes the given function and retries it if it
// returns an Aborted error. The delay between retries is the delay returned
// by Cloud Spanner, and if none is returned, the calculated delay with a
// minimum of 10ms and maximum of 32s.
func runWithRetryOnAborted(ctx context.Context, f func(context.Context) error) error {
	retryer := onCodes(DefaultRetryBackoff, codes.Aborted)
	funcWithRetry := func(ctx context.Context) error {
		for {
			err := f(ctx)
			if err == nil {
				return nil
			}
			delay, shouldRetry := retryer.Retry(err)
			if !shouldRetry {
				return err
			}
			trace.TracePrintf(ctx, nil, "Backing off after ABORTED for %s, then retrying", delay)
			if err := gax.Sleep(ctx, delay); err != nil {
				return err
			}
		}
	}
	return funcWithRetry(ctx)
}

// extractRetryDelay extracts retry backoff if present.
func extractRetryDelay(err error) (time.Duration, bool) {
	trailers := errTrailers(err)
	if trailers == nil {
		return 0, false
	}
	elem, ok := trailers[retryInfoKey]
	if !ok || len(elem) <= 0 {
		return 0, false
	}
	_, b, err := metadata.DecodeKeyValue(retryInfoKey, elem[0])
	if err != nil {
		return 0, false
	}
	var retryInfo edpb.RetryInfo
	if proto.Unmarshal([]byte(b), &retryInfo) != nil {
		return 0, false
	}
	delay, err := ptypes.Duration(retryInfo.RetryDelay)
	if err != nil {
		return 0, false
	}
	return delay, true
}
