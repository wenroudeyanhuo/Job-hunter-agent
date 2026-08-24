# Multi-Agent and Eino Roadmap

Job Hunter Agent is moving from a single assistant workflow toward a multi-agent recruiting employee.

## Current Multi-Agent Team

The backend now defines four specialized agents in `backend/internal/jobs/multi_agent_runtime.go`:

- `Source Scout`: watches source health and proposes source discovery or repair.
- `Job Analyst`: reviews tracked jobs, strong matches, and manual decisions.
- `Memory Keeper`: checks semantic memory coverage and proposes memory maintenance.
- `Planner`: turns observations into approval-gated workflow actions.

The current runner is deterministic. This keeps local development predictable while the role contracts, actions, and guardrails stabilize.

## Eino Boundary

The orchestration metadata is marked as `eino_ready` because the role contracts already map to Eino concepts:

- Each specialized agent can become an Eino ADK specialist or a graph node.
- The current sequential flow can become an Eino Graph or WorkflowAgent.
- The existing approval queue remains the human-in-the-loop gate before external actions.
- Semantic memory stays behind the repository API so it can later use local hash embeddings, model embeddings, pgvector, Qdrant, or another vector backend.

## Target Eino Pattern

Recommended first Eino migration:

1. Keep the existing deterministic `RunRecruitingAgentCycle` as the fallback runner.
2. Add an `EinoRecruitingOrchestrator` behind the same input/output structs.
3. Start with a preset sequential graph:
   `SourceScout -> JobAnalyst -> MemoryKeeper -> Planner`.
4. Add interrupts around action execution so approvals continue to happen in the current action-request queue.
5. Move to Agent-as-Tool only after the deterministic graph is stable and model-backed delegation is useful.

## Guardrails

- No automatic resume submission.
- No login automation without explicit user approval.
- No captcha or anti-bot bypass.
- External actions must remain approval-gated.
- Memory must not store secrets.
