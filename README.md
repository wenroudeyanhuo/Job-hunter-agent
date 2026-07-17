# Job Hunter Agent

Job Hunter Agent is a local-first assistant for collecting, filtering, scoring, and tracking campus recruitment opportunities.

The project is currently focused on Shenzhen-oriented technical roles such as frontend, backend, Java, Go, algorithm, and AI application development. It is designed as a practical personal workflow tool first, with a clean path toward more capable recruiting automation later.

## Status

Early MVP. The current version provides a Go backend foundation, SQLite persistence, scoring and deduplication rules, a crawl runner, scheduled runs, Feishu webhook notification support, and a React dashboard shell.

## Features

- Local SQLite database for job opportunities and crawl logs.
- Go backend with REST APIs for jobs, crawl runs, settings, and Feishu test notification.
- Rule-based scoring for target cities, target roles, company signals, campus recruitment signals, and application links.
- Hard filters for obvious outsourcing, training/course-sales content, unclear-conversion internships, and unrelated roles.
- Deduplication by application URL and normalized company/title/city.
- Scheduled crawl runner for 09:00, 12:00, and 18:00.
- React dashboard for reviewing jobs, filtering by status/direction, updating status, and running a crawl.

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

## Project Structure

```text
.
+-- backend
|   +-- cmd/server              # Backend entrypoint
|   +-- internal/app            # Application wiring
|   +-- internal/config         # Environment configuration
|   +-- internal/crawl          # Crawl runner and scheduler
|   +-- internal/db             # SQLite schema and connection
|   +-- internal/domain         # Shared domain types
|   +-- internal/http           # REST API handlers and routes
|   +-- internal/jobs           # Job repository, scoring, and dedupe
|   +-- internal/notify         # Feishu webhook notification
+-- frontend
    +-- src                     # React dashboard
```

## Development

### Environment

Copy `.env.example` to `.env` if you want to keep local settings in a file. The app reads these environment variables:

```env
APP_ADDR=:8080
APP_DB_PATH=data/job-hunter-agent.db
FEISHU_WEBHOOK_URL=
DISABLE_SCHEDULER=0
```

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

The frontend dev server proxies `/api` and `/healthz` to `http://localhost:8080`.

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

## Roadmap

- Add manual URL import API and dashboard flow.
- Add the first real public-source collector.
- Add source configuration in the dashboard.
- Improve parsing for company, role, city, deadline, and application URL.
- Add Feishu summary sending after crawl runs.
- Add job detail view with notes and application metadata.
- Add optional Feishu Base or spreadsheet sync.
- Explore resume matching and assisted application workflows after the collection pipeline is reliable.

## Contributing

This project is early and evolving. Small, focused pull requests are preferred. Please avoid adding automation that logs in to third-party platforms, bypasses anti-bot systems, or submits applications without explicit user confirmation.

## License

MIT
