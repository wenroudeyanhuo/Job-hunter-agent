# Open Source Setup Guide

This project is local-first by default. A new user should be able to run it without cloud services, then opt into Feishu and DeepSeek/OpenAI-compatible model features.

## Quick Start

1. Copy `.env.example` to `.env`.
2. Adjust `APP_DB_PATH`, `SOURCE_URLS`, and optional notification/model keys.
3. Run the backend:

```powershell
cd backend
go run ./cmd/server
```

4. Run the frontend:

```powershell
cd frontend
npm install
npm run dev
```

5. Open `http://localhost:5173`.

## Docker Compose

```powershell
docker compose up --build
```

The backend stores SQLite data in the `job-hunter-data` volume. Keep that volume if you want the employee's memory, cycles, jobs, and decisions to persist.

Recommended personal long-running setup:

```powershell
copy .env.example .env
docker compose up -d --build
docker compose logs -f backend
```

Use `docker compose down` to stop the service while keeping data. Avoid `docker compose down -v` unless you intentionally want to remove the SQLite database volume.

SQLite backup example:

```powershell
docker compose exec backend sh -lc "cp /data/job-hunter-agent.db /data/job-hunter-agent.db.backup"
```

Optional Qdrant semantic memory:

```powershell
$env:SEMANTIC_MEMORY_PROVIDER="qdrant"
$env:QDRANT_URL="http://qdrant:6333"
docker compose --profile qdrant up --build
```

The default `local_hash` provider remains zero-config. Qdrant is useful when a personal deployment wants external vector search while keeping SQLite as the local system of record.

## Optional Feishu

Set `FEISHU_WEBHOOK_URL` or paste a webhook in Settings. Dashboard settings take priority over the environment variable.

Feishu reports include the duty report and, when available, the latest Agent Cycle summary so users can see what the digital employee just decided.

## Optional DeepSeek

For DeepSeek:

```env
LLM_PROVIDER=deepseek
DEEPSEEK_API_KEY=your_key
DEEPSEEK_MODEL=deepseek-chat
```

The backend treats DeepSeek as an OpenAI-compatible chat-completions endpoint and falls back to local rules if the model is unavailable.

## Optional Eino Graph

The default agent orchestration is deterministic so the project works without optional dependencies. Users who want to try the Eino Graph path can install Eino and run the backend with the `eino` build tag:

```powershell
.\scripts\run-eino.ps1
```

Tagged verification:

```powershell
go test -tags eino ./internal/jobs -run TestEinoRecruitingOrchestratorRunsGraph
```

For a resume-facing mapping of claims to code and tests, see [docs/resume-proof.md](resume-proof.md).

## Scheduler Requirements

The scheduler requires a long-lived backend process. It handles:

- scheduled crawls,
- automatic source discovery,
- stale-task escalation,
- automatic duty reports,
- follow-up Agent Cycles after scheduled work.

Static hosting such as Vercel is suitable for the frontend only. The backend needs persistent storage and an always-on process, or a hosted worker/database setup.

For personal use, keep the backend process running through Docker Compose, a systemd service, or a small VPS process manager. Static frontend hosting alone is not enough because the scheduler, crawler, SQLite database, Feishu reports, and model calls live in the backend.

## Safety Boundaries

- No automatic resume submission.
- No login automation without explicit user approval.
- No captcha or anti-bot bypass.
- External actions stay in the approval queue.
- Secrets should stay in `.env` or Settings and should not be stored in semantic memory.
