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
cd backend
go get github.com/cloudwego/eino@latest
$env:AGENT_ORCHESTRATOR="eino_graph"
go run -tags eino ./cmd/server
```

Tagged verification:

```powershell
go test -tags eino ./internal/jobs -run TestEinoRecruitingOrchestratorRunsGraph
```

## Scheduler Requirements

The scheduler requires a long-lived backend process. It handles:

- scheduled crawls,
- automatic source discovery,
- stale-task escalation,
- automatic duty reports,
- follow-up Agent Cycles after scheduled work.

Static hosting such as Vercel is suitable for the frontend only. The backend needs persistent storage and an always-on process, or a hosted worker/database setup.

## Safety Boundaries

- No automatic resume submission.
- No login automation without explicit user approval.
- No captcha or anti-bot bypass.
- External actions stay in the approval queue.
- Secrets should stay in `.env` or Settings and should not be stored in semantic memory.
