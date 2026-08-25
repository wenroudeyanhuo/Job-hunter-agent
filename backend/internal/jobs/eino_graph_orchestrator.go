//go:build eino

package jobs

import (
	"context"
	"time"

	"github.com/cloudwego/eino/compose"
)

const (
	einoNodeInit         = "init_cycle"
	einoNodeSourceScout  = "source_scout"
	einoNodeJobAnalyst   = "job_analyst"
	einoNodeMemoryKeeper = "memory_keeper"
	einoNodePlanner      = "planner"
	einoNodeToolPlanner  = "tool_planner"
	einoNodeObserver     = "observer"
	einoNodeFinalize     = "finalize_cycle"
)

type einoCycleState struct {
	Input MultiAgentCycleInput
	Cycle MultiAgentCycle
}

type einoRecruitingOrchestrator struct {
	invoke func(context.Context, MultiAgentCycleInput) (MultiAgentCycle, error)
}

func NewEinoRecruitingOrchestrator(ctx context.Context) (RecruitingOrchestrator, error) {
	graph := compose.NewGraph[MultiAgentCycleInput, MultiAgentCycle]()
	if err := graph.AddLambdaNode(einoNodeInit, compose.InvokableLambda(initEinoCycle)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeSourceScout, compose.InvokableLambda(runEinoSourceScout)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeJobAnalyst, compose.InvokableLambda(runEinoJobAnalyst)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeMemoryKeeper, compose.InvokableLambda(runEinoMemoryKeeper)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodePlanner, compose.InvokableLambda(runEinoPlanner)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeToolPlanner, compose.InvokableLambda(runEinoToolPlanner)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeObserver, compose.InvokableLambda(runEinoObserver)); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode(einoNodeFinalize, compose.InvokableLambda(finalizeEinoCycle)); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(compose.START, einoNodeInit); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeInit, einoNodeSourceScout); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeSourceScout, einoNodeJobAnalyst); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeJobAnalyst, einoNodeMemoryKeeper); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeMemoryKeeper, einoNodePlanner); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodePlanner, einoNodeToolPlanner); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeToolPlanner, einoNodeObserver); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeObserver, einoNodeFinalize); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(einoNodeFinalize, compose.END); err != nil {
		return nil, err
	}
	runner, err := graph.Compile(ctx)
	if err != nil {
		return nil, err
	}
	return &einoRecruitingOrchestrator{
		invoke: func(ctx context.Context, input MultiAgentCycleInput) (MultiAgentCycle, error) {
			return runner.Invoke(ctx, input)
		},
	}, nil
}

func (o *einoRecruitingOrchestrator) Run(input MultiAgentCycleInput) MultiAgentCycle {
	cycle, err := o.invoke(context.Background(), input)
	if err != nil {
		return RunRecruitingAgentCycle(input)
	}
	cycle.Team.Orchestrator.Provider = "eino_graph"
	cycle.Team.Orchestrator.Pattern = "plan_tool_approval_observe_replan"
	cycle.Team.Orchestrator.NextStep = "Running through an Eino Graph with memory, tool planning, approval boundary, and observer nodes."
	return cycle
}

func initEinoCycle(_ context.Context, input MultiAgentCycleInput) (einoCycleState, error) {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return einoCycleState{
		Input: input,
		Cycle: MultiAgentCycle{
			GeneratedAt: now,
			Team:        BuildRecruitingAgentTeam(),
		},
	}, nil
}

func runEinoSourceScout(_ context.Context, state einoCycleState) (einoCycleState, error) {
	state.Cycle.Trace = append(state.Cycle.Trace, runSourceScoutAgent(state.Input))
	return state, nil
}

func runEinoJobAnalyst(_ context.Context, state einoCycleState) (einoCycleState, error) {
	state.Cycle.Trace = append(state.Cycle.Trace, runJobAnalystAgent(state.Input))
	return state, nil
}

func runEinoMemoryKeeper(_ context.Context, state einoCycleState) (einoCycleState, error) {
	state.Cycle.Trace = append(state.Cycle.Trace, runMemoryKeeperAgent(state.Input))
	return state, nil
}

func runEinoPlanner(_ context.Context, state einoCycleState) (einoCycleState, error) {
	state.Cycle.Trace = append(state.Cycle.Trace, runPlannerAgent(state.Input))
	return state, nil
}

func runEinoToolPlanner(_ context.Context, state einoCycleState) (einoCycleState, error) {
	for _, trace := range state.Cycle.Trace {
		state.Cycle.Actions = appendUniqueAgentActions(state.Cycle.Actions, trace.Actions...)
	}
	state.Cycle.AutonomyPlan = BuildAgentAutonomyPlan(state.Cycle.Actions, "eino_graph")
	state.Cycle.Trace = append(state.Cycle.Trace, MultiAgentTrace{
		AgentKey:    MultiAgentPlanner,
		Observation: "Registered tools available: " + ModelActionPromptList(),
		Decision:    "Built a structured tool plan with approval gates before every executable step.",
	})
	return state, nil
}

func runEinoObserver(_ context.Context, state einoCycleState) (einoCycleState, error) {
	state.Cycle.Trace = append(state.Cycle.Trace, MultiAgentTrace{
		AgentKey:    MultiAgentObserver,
		Observation: "Observer node is waiting for approved tool execution receipts.",
		Decision:    "After execution, summarize the receipt, update memory or tasks, then trigger the next planning pass.",
	})
	return state, nil
}

func finalizeEinoCycle(_ context.Context, state einoCycleState) (MultiAgentCycle, error) {
	for _, trace := range state.Cycle.Trace {
		state.Cycle.Actions = appendUniqueAgentActions(state.Cycle.Actions, trace.Actions...)
	}
	if state.Cycle.AutonomyPlan.Mode == "" {
		state.Cycle.AutonomyPlan = BuildAgentAutonomyPlan(state.Cycle.Actions, "eino_graph")
	}
	state.Cycle.ReadinessScore = multiAgentReadinessScore(state.Input, state.Cycle.Actions)
	state.Cycle.Summary = buildMultiAgentCycleSummary(state.Input, state.Cycle)
	return state.Cycle, nil
}
