package otlp

import (
	"context"
	"fmt"

	"github.com/thread_koder/mochi/core/internal/span"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

func ingestTraces(ctx context.Context, traces ptrace.Traces) (ptraceotlp.ExportResponse, error) {
	mapped := mapTraces(ctx, traces)
	if len(mapped.Spans) > 0 {
		if err := span.Ingest(ctx, mapped.Spans); err != nil {
			return ptraceotlp.ExportResponse{}, fmt.Errorf("span ingest: %w", err)
		}
	}
	resp := ptraceotlp.NewExportResponse()
	if mapped.Rejected > 0 {
		partial := resp.PartialSuccess()
		partial.SetRejectedSpans(mapped.Rejected)
		partial.SetErrorMessage(mapped.Message)
	}
	return resp, nil
}
