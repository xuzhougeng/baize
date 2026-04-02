package ai

import (
	"context"
	"fmt"
	"strings"
)

// ToolExecutionResult is the normalized tool result returned by the app runtime.
// RawOutput stays in scratchpad/debug state, while OutputSummary is the planner-facing form.
type ToolExecutionResult struct {
	RawOutput     string
	OutputSummary string
}

// ToolAttempt records a single tool call made during the agent loop.
type ToolAttempt struct {
	ToolName      string
	ToolInput     string
	RawOutput     string
	OutputSummary string
	Succeeded     bool
	FailureReason string
}

// AgentTaskState holds all mutable state for a running agent task.
type AgentTaskState struct {
	Goal            string
	UserInput       string
	HistorySummary  string
	WorkingSummary  string
	FinalSummary    string
	Observations    []string
	ToolAttempts    []ToolAttempt
	CandidateTools  []AgentToolDefinition
	PendingQuestion string
	LastAnswerDraft string
	SideEffects     []string
}

// LoopAction is the action the planner requests at each step.
type LoopAction string

const (
	LoopContinue LoopAction = "continue"
	LoopAnswer   LoopAction = "answer"
	LoopAsk      LoopAction = "ask"
	LoopStop     LoopAction = "stop"
)

// LoopDecision is the planner's response at each step.
type LoopDecision struct {
	Action    LoopAction
	ToolName  string
	ToolInput string
	Answer    string
	Question  string
	Reason    string
}

// AgentLoopRuntime is implemented by the app layer.
// Uses any for mc to avoid import cycle.
type AgentLoopRuntime interface {
	LoadHistory(ctx context.Context, mc any) []ConversationMessage
	ListTools(ctx context.Context, mc any) ([]AgentToolDefinition, error)
	ExecuteTool(ctx context.Context, mc any, toolName, toolInput string) (ToolExecutionResult, error)
	UpdateWorkingSummary(ctx context.Context, mc any, summary string)
	PersistTurn(ctx context.Context, mc any, userInput, assistantReply, finalSummary string)
}

// AgentPlanner is implemented by the ai.Service.
type AgentPlanner interface {
	PlanNext(ctx context.Context, task string, history []ConversationMessage, tools []AgentToolDefinition, state AgentTaskState) (LoopDecision, error)
	SummarizeWorkingState(ctx context.Context, state AgentTaskState) (string, error)
	SummarizeFinal(ctx context.Context, state AgentTaskState, finalAnswer string) (string, error)
}

// RunAgentLoop runs the phase-based agent loop.
func RunAgentLoop(ctx context.Context, runtime AgentLoopRuntime, planner AgentPlanner, mc any, userInput string, maxSteps int) (finalAnswer string, finalSummary string, err error) {
	history := runtime.LoadHistory(ctx, mc)
	tools, err := runtime.ListTools(ctx, mc)
	if err != nil {
		return "", "", fmt.Errorf("listing tools: %w", err)
	}

	userInput = strings.TrimSpace(userInput)
	state := AgentTaskState{
		Goal:           userInput,
		UserInput:      userInput,
		CandidateTools: tools,
	}

	finalize := func(reply string) (string, string, error) {
		state.FinalSummary, _ = planner.SummarizeFinal(ctx, state, reply)
		runtime.PersistTurn(ctx, mc, userInput, reply, state.FinalSummary)
		return reply, state.FinalSummary, nil
	}

	for step := 0; step < maxSteps; step++ {
		decision, err := planner.PlanNext(ctx, userInput, history, tools, state)
		if err != nil {
			return "", "", fmt.Errorf("planning step %d: %w", step, err)
		}

		switch decision.Action {
		case LoopContinue:
			result, execErr := runtime.ExecuteTool(ctx, mc, decision.ToolName, decision.ToolInput)
			attempt := ToolAttempt{
				ToolName:      decision.ToolName,
				ToolInput:     decision.ToolInput,
				RawOutput:     strings.TrimSpace(result.RawOutput),
				OutputSummary: strings.TrimSpace(result.OutputSummary),
				Succeeded:     execErr == nil,
			}
			if execErr != nil {
				attempt.FailureReason = execErr.Error()
			}
			state.ToolAttempts = append(state.ToolAttempts, attempt)
			if workingSummary, summarizeErr := planner.SummarizeWorkingState(ctx, state); summarizeErr == nil {
				state.WorkingSummary = strings.TrimSpace(workingSummary)
			}
			runtime.UpdateWorkingSummary(ctx, mc, state.WorkingSummary)

		case LoopAnswer:
			return finalize(decision.Answer)

		case LoopAsk:
			return finalize(decision.Question)

		case LoopStop:
			return finalize(decision.Reason)

		default:
			return "", "", fmt.Errorf("unsupported loop action %q", decision.Action)
		}
	}

	return finalize("任务达到最大步骤限制，先在这里停止。")
}
