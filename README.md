# Job Hunter Agent

A local-first job hunting assistant for collecting, filtering, and tracking campus recruitment opportunities.

## Status

Early MVP. The current version provides a Go backend foundation, SQLite persistence, scoring and deduplication rules, a crawl runner, scheduled runs, Feishu webhook notification support, and a React dashboard shell.

## What It Does

- Tracks campus recruitment and job-hunting opportunities in a local SQLite database.
- Scores jobs for Shenzhen-focused frontend, backend, Java, Go, algorithm, and AI application development roles.
- Filters obvious outsourcing, training, low-quality, and unclear-conversion internship content.
- Provides a local dashboard for reviewing jobs and updating status.
- Supports manual crawl runs and scheduled runs at 09:00, 12:00, and 18:00.
- Can send Feishu incoming webhook notifications.

## What It Does Not Do

- It does not automatically submit resumes.
- It does not log in to job platforms.
- It does not bypass captcha, sliders, or anti-bot systems.
- It does not sync to Feishu Base yet.

## Development

### Environment

Copy `.env.example` to `.env` if you want to keep local settings in a file. The app reads these environment variables:

```env
APP_ADDR=:8080
APP_DB_PATH=data/job-hunter-agent.db
FEISHU_WEBHOOK_URL=
DISABLE_SCHEDULER=0
SOURCE_URLS=
```

`SOURCE_URLS` can contain comma-separated or newline-separated public recruitment URLs. Manual and scheduled crawl runs import these URLs, score them, deduplicate them, and store them in the local database.

### Backend

Requires Go 1.25 or newer.

```bash
cd backend
go run ./cmd/server
```

The backend listens on `http://localhost:8080` by default.

Useful commands:

```bash
cd backend
go test ./...
```

### Frontend

Requires Node.js and npm.

```bash
cd frontend
npm install
npm run dev
```

The dashboard is available at `http://localhost:5173` by default. The frontend dev server proxies `/api` and `/healthz` to the backend at `http://localhost:8080`.

Build check:

```bash
cd frontend
npm run build
```

## Local Data

By default, the backend stores SQLite data under:

```text
backend/data/job-hunter-agent.db
```

Local databases, logs, build outputs, private planning docs, and environment files are ignored by Git.
