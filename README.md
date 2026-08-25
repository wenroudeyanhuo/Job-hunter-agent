# Job Hunter Agent / 秋招数字员工

![Job Hunter Agent avatar](frontend/public/assets/job-agent-avatar.png)

[中文文档](README.zh-CN.md) | [English README](README.en.md)

Job Hunter Agent 是一个面向秋招/校招场景的本地优先数字员工项目。它会持续发现招聘来源、采集岗位、过滤低质量信息、为岗位打分、生成每日待办，并通过网页看板和飞书机器人把“今天该处理什么”推给你。

Job Hunter Agent is a local-first recruiting digital employee. It discovers recruiting sources, collects and scores jobs, generates daily work queues, sends Feishu reports, and is evolving toward a multi-agent workflow with optional model and Eino Graph orchestration.

## Highlights / 核心亮点

- Go backend, React dashboard, SQLite local persistence.
- Public source discovery, validation, crawling, deduplication, filtering, and scoring.
- Candidate profile, job details, application Kanban, daily task queue, and Feishu reports.
- Global digital employee chat with local-rule fallback and optional DeepSeek/OpenAI-compatible model mode.
- Multi-agent runtime with Source Scout, Job Analyst, Memory Keeper, Planner, Observer, and an optional Eino Graph path.
- Approval-gated Agent Tool Registry so suggested actions are reviewed before execution.
- Local semantic memory by default, with optional Qdrant vector search for personal deployments and provider hooks for future DeepSeek embeddings / pgvector.

## Quick Start / 快速开始

Backend:

```bash
cd backend
go run ./cmd/server
```

Frontend:

```bash
cd frontend
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

Docker Compose:

```bash
docker compose up --build
```

## Documentation / 文档

- [中文文档](README.zh-CN.md)
- [English README](README.en.md)
- [Open source setup guide](docs/open-source-setup.md)
- [Multi-Agent and Eino roadmap](docs/multi-agent-eino-roadmap.md)
- [数字员工产品化路线图](docs/product-agent-roadmap.zh-CN.md)

## Safety Boundary / 安全边界

- No automatic resume submission.
- No login automation without explicit user approval.
- No captcha, slider, or anti-bot bypass.
- External actions must remain approval-gated.
- Memory must not store secrets.

## License

MIT
