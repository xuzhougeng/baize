package ai

import (
	"context"
	"strings"
	"sync"
)

type callTraceCollector struct {
	mu        sync.Mutex
	steps     []CallTraceStep
	observers []func(CallTraceStep)
}

type callTraceCollectorKey struct{}

func WithCallTraceCollector(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Value(callTraceCollectorKey{}) != nil {
		return ctx
	}
	return context.WithValue(ctx, callTraceCollectorKey{}, &callTraceCollector{})
}

func WithCallTraceObserver(ctx context.Context, observer func(CallTraceStep)) context.Context {
	if observer == nil {
		return ctx
	}
	ctx = WithCallTraceCollector(ctx)
	collector, _ := ctx.Value(callTraceCollectorKey{}).(*callTraceCollector)
	if collector == nil {
		return ctx
	}
	collector.mu.Lock()
	collector.observers = append(collector.observers, observer)
	collector.mu.Unlock()
	return ctx
}

func CallTraceFromContext(ctx context.Context) []CallTraceStep {
	if ctx == nil {
		return nil
	}
	collector, _ := ctx.Value(callTraceCollectorKey{}).(*callTraceCollector)
	if collector == nil {
		return nil
	}
	collector.mu.Lock()
	defer collector.mu.Unlock()
	out := make([]CallTraceStep, 0, len(collector.steps))
	for _, step := range collector.steps {
		title := strings.TrimSpace(step.Title)
		detail := strings.TrimSpace(step.Detail)
		if title == "" && detail == "" {
			continue
		}
		out = append(out, CallTraceStep{
			Title:  title,
			Detail: detail,
		})
	}
	return out
}

func AddCallTraceStep(ctx context.Context, step CallTraceStep) {
	if ctx == nil {
		return
	}
	collector, _ := ctx.Value(callTraceCollectorKey{}).(*callTraceCollector)
	if collector == nil {
		return
	}
	step.Title = strings.TrimSpace(step.Title)
	step.Detail = strings.TrimSpace(step.Detail)
	if step.Title == "" && step.Detail == "" {
		return
	}
	collector.mu.Lock()
	collector.steps = append(collector.steps, step)
	observers := append([]func(CallTraceStep){}, collector.observers...)
	collector.mu.Unlock()
	for _, observer := range observers {
		if observer != nil {
			observer(step)
		}
	}
}
