package ai

import (
	"context"
	"testing"
)

func TestWithCallTraceObserverReceivesStepsImmediately(t *testing.T) {
	t.Parallel()

	ctx := WithCallTraceObserver(context.Background(), nil)
	var observed []CallTraceStep
	ctx = WithCallTraceObserver(ctx, func(step CallTraceStep) {
		observed = append(observed, step)
	})

	AddCallTraceStep(ctx, CallTraceStep{Title: "AI 路由", Detail: "command=answer"})
	AddCallTraceStep(ctx, CallTraceStep{Title: "执行模式", Detail: "mode=agent"})

	if len(observed) != 2 {
		t.Fatalf("expected 2 observed steps, got %#v", observed)
	}
	if observed[0].Title != "AI 路由" || observed[1].Title != "执行模式" {
		t.Fatalf("unexpected observed steps: %#v", observed)
	}

	stored := CallTraceFromContext(ctx)
	if len(stored) != 2 {
		t.Fatalf("expected stored steps, got %#v", stored)
	}
}
