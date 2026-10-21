// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"testing"

	otelog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
)

// resourceRecorder captures the resource of every log record it receives.
type resourceRecorder struct {
	attrs map[string]bool
}

func (r *resourceRecorder) OnEmit(_ context.Context, record *sdklog.Record) error {
	res := record.Resource()
	r.attrs = make(map[string]bool, res.Len())
	for _, kv := range res.Attributes() {
		r.attrs[string(kv.Key)] = true
	}
	return nil
}

func (r *resourceRecorder) Shutdown(context.Context) error   { return nil }
func (r *resourceRecorder) ForceFlush(context.Context) error { return nil }

// TestLogProviderUsesInitializedResource checks that the log provider carries
// the same resource as the trace and metric providers. initResource() adds the
// OS, process, container and host attributes; the default resource adds the
// telemetry SDK attributes. A log record must carry all of them so logs join
// traces and metrics on the same resource.
func TestLogProviderUsesInitializedResource(t *testing.T) {
	rec := &resourceRecorder{}
	lp := newLogProvider(rec)
	lp.Logger("productcatalogservice").Emit(context.Background(), otelog.Record{})

	if len(rec.attrs) == 0 {
		t.Fatal("log record carries no resource attributes")
	}
	for _, kv := range initResource().Attributes() {
		if !rec.attrs[string(kv.Key)] {
			t.Errorf("log record resource is missing %q from initResource()", kv.Key)
		}
	}
	for _, key := range []string{"host.name", "os.type", "process.pid", "telemetry.sdk.language"} {
		if !rec.attrs[key] {
			t.Errorf("log record resource is missing %q", key)
		}
	}
}
