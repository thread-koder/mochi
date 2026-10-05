package span

import (
	"context"
	"fmt"
	"time"

	"github.com/thread_koder/mochi/core/internal/database"
)

func GetNamespaceSpans(ctx context.Context, namespace string, since time.Time, limit int) (SpansResponse, error) {
	rows, err := database.GetSpansByNamespace(ctx, namespace, since, clampLimit(limit))
	if err != nil {
		return SpansResponse{}, fmt.Errorf("get namespace spans: %w", err)
	}
	return toResponse(rows), nil
}

func GetWorkloadSpans(ctx context.Context, kind, namespace, name string, since time.Time, limit int) (SpansResponse, error) {
	rows, err := database.GetSpansByWorkload(ctx, kind, namespace, name, since, clampLimit(limit))
	if err != nil {
		return SpansResponse{}, fmt.Errorf("get workload spans: %w", err)
	}
	return toResponse(rows), nil
}

func toResponse(rows []*database.Span) SpansResponse {
	result := make([]SpanDTO, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDTO(row))
	}
	return SpansResponse{Spans: result}
}
