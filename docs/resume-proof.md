# Resume Proof Checklist

This document maps the resume-facing claims of TalentPilot / Job Hunter Agent to concrete code paths, configuration switches, and verification commands.

## Multi-Agent Runtime

Resume claim: Eino-based Multi-Agent workflow with Source Scout, Job Analyst, Memory Keeper, Planner, Observer, and auditable Agent Cycle records.

Project evidence:

- `backend/internal/jobs/multi_agent_runtime.go` defines the specialist agents and deterministic orchestration contract.
- `backend/internal/jobs/eino_graph_orchestrator.go` maps the specialist flow into an optional Eino Graph behind the `eino` build tag.
- `backend/internal/jobs/agent_cycle.go` persists cycle metadata, observations, plan steps, action proposals, and re-plan guidance.
- `backend/internal/jobs/orchestrator_config_eino.go` enables `AGENT_ORCHESTRATOR=eino_graph` when the binary is built with `-tags eino`.

Verify:

```powershell
cd backend
go test -tags eino ./internal/jobs -run "Test(EinoRecruitingOrchestratorRunsGraph|ConfiguredRecruitingOrchestratorUsesEinoGraph)" -count=1
```

Run locally through Eino Graph:

```powershell
.\scripts\run-eino.ps1
```

## Qdrant Semantic Memory

Resume claim: Qdrant vector retrieval is available for profile-aware recommendation reasons and memory-backed ranking.

Project evidence:

- `backend/internal/jobs/semantic_memory.go` keeps semantic memory behind provider boundaries.
- `backend/internal/jobs/qdrant_memory.go` implements the external Qdrant provider.
- `backend/internal/jobs/semantic_memory_test.go` verifies the repository uses Qdrant when `SEMANTIC_MEMORY_PROVIDER=qdrant` and `QDRANT_URL` are configured.
- `docker-compose.yml` includes an optional Qdrant profile for personal deployments.

Run with Qdrant:

```powershell
$env:SEMANTIC_MEMORY_PROVIDER="qdrant"
$env:QDRANT_URL="http://qdrant:6333"
docker compose --profile qdrant up --build
```

## Real Job Collection And Quality

Resume claim: public web parsing, JSON-LD extraction, source validation, normalization, deduplication, scoring, and low-quality filtering.

Project evidence:

- `backend/internal/jobs/public_url_crawler.go` handles public URL collection, timeouts, retries, source health updates, and landing-page filtering.
- `backend/internal/jobs/importer.go` normalizes imported jobs and stores parser gap metadata.
- `backend/internal/jobs/scorer.go` scores jobs with settings and decision feedback.
- `backend/internal/jobs/source_discovery.go` expands the source pool from curated and discovered sources.

Verify:

```powershell
cd backend
go test ./...
```

## Tool Registry And HITL Approval

Resume claim: Tool Registry plus human-in-the-loop approval queue for risky actions.

Project evidence:

- `backend/internal/jobs/agent_tools.go` exposes agent tool definitions, approval requirements, and execution contracts.
- `backend/internal/jobs/agent_actions.go` stores approval requests and execution status.
- `backend/internal/jobs/agent_command.go` routes user commands through model/local planning and safe actions.

## DeepSeek / OpenAI-Compatible Dialogue

Resume claim: DeepSeek/OpenAI-compatible model integration with local deterministic fallback.

Project evidence:

- `backend/internal/jobs/llm_config.go` stores and health-checks model provider configuration.
- `backend/internal/jobs/llm_client.go` uses OpenAI-compatible chat completion APIs.
- `backend/internal/jobs/agent_chat.go` grounds chat in jobs, memory, preferences, and recent agent state.
- The frontend model healthcheck is available in the employee chat panel and Settings health area.

## Feishu, Docker, And Local Long-Running Use

Resume claim: Feishu Webhook reports, Docker Compose, and local-first long-running operation.

Project evidence:

- `backend/internal/jobs/feishu.go` sends reports to Feishu webhook.
- `backend/internal/jobs/scheduler.go` runs crawl, source discovery, task escalation, and daily duty report jobs.
- `docker-compose.yml` starts backend, frontend, and optional Qdrant with persistent volumes.
- `docs/open-source-setup.md` documents Docker Compose, backup, Feishu, model, and long-running options.

## Baseline Verification

Use this before making a release or quoting the project in a resume:

```powershell
cd backend
go test ./...
go test -tags eino ./internal/jobs -run "Test(EinoRecruitingOrchestratorRunsGraph|ConfiguredRecruitingOrchestratorUsesEinoGraph)" -count=1

cd ..\frontend
npm run build
```
