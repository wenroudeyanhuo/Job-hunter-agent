# Job Hunter Agent

Job Hunter Agent is a local-first assistant for collecting, filtering, scoring, and tracking campus recruitment opportunities.

The project is currently focused on Shenzhen-oriented technical roles such as frontend, backend, Java, Go, algorithm, and AI application development. It is designed as a practical personal workflow tool first, with a clean path toward more capable recruiting automation later.

## Status

Early MVP. The current version provides a Go backend foundation, SQLite persistence, scoring and deduplication rules, a crawl runner, scheduled runs, source discovery, Feishu webhook notification support, optional model-backed chat, and a React dashboard shell.

## Features

- Local SQLite database for job opportunities and crawl logs.
- Go backend with REST APIs for jobs, crawl runs, settings, and Feishu test notification.
- Rule-based scoring for target cities, target roles, company signals, campus recruitment signals, and application links.
- Hard filters for obvious outsourcing, training/course-sales content, unclear-conversion internships, and unrelated roles.
- Deduplication by application URL and normalized company/title/city.
- Scheduled crawl runner for 09:00, 12:00, and 18:00.
- React dashboard for reviewing jobs, filtering by status/direction, updating status, and running a crawl.
- Candidate profile page for cities, directions, skills, education, preferred companies, blocked keywords, and notes.
- Job detail panel with profile-aware fit signals, risks, suggested action, notes, and decision history.
- Application workspace for turning interested strong matches into human-approved application preparation plans.
- Daily agent task queue generated from recommended jobs, manual decisions, source issues, and crawl history.
- Digital employee sidebar with an agent profile, avatar, maturity score, capability map, operating cycle, and mainstream capability gaps.
- Command Center for rule-based natural-language workflow commands such as changing target cities/directions, refreshing tasks, running a crawl, and sending Feishu reports.
- Global digital employee chat with a persistent 3D avatar, local rule fallback, saved chat history, and optional OpenAI-compatible model mode.
- Source discovery that proposes broader official, community, and job-platform search entrances from the user's target cities and directions.
- Source-candidate validation that fetches candidate pages, checks recruitment signals and discovered job links, then adjusts confidence before the source is accepted.
- Automatic duty report controls with configurable report time, scheduler tick, task SLA, stale-task detection, escalation, snooze, completion reasons, and last-sent tracking.
- Automatic source discovery controls with a configurable interval so the assistant can keep expanding the source pool over time.
- Feishu webhook summaries after crawl runs when a webhook is configured in Settings or `FEISHU_WEBHOOK_URL`.

## What It Does

- Tracks campus recruitment and job-hunting opportunities in a local SQLite database.
- Scores jobs for Shenzhen-focused frontend, backend, Java, Go, algorithm, and AI application development roles.
- Filters obvious outsourcing, training, low-quality, and unclear-conversion internship content.
- Provides a local dashboard for reviewing jobs and updating status.
- Builds a local candidate profile and uses it to explain why a role fits or carries risk.
- Records job decisions such as interested, applied, ignored, and notes updates as a timeline.
- Prepares application plans for interested strong matches, including priority, target date, checklist, and next action.
- Generates a daily task queue for recommended jobs, human decisions, unhealthy sources, and crawl setup.
- Shows what the assistant can already do, where it is weaker than mainstream digital employees, and which capability should be improved next.
- Accepts simple workflow commands from the digital employee sidebar. Current parsing is deterministic and transparent, not LLM-based.
- Keeps a global chat assistant available across pages. Without a model key it answers with local recruiting context; with model settings it calls an OpenAI-compatible chat-completions endpoint and falls back locally if the model fails.
- Discovers and validates new source candidates so the crawl pool does not stay fixed forever.
- Tracks stale or escalated daily tasks, supports snoozing or closing work items with reasons, and can send an automatic duty report when enabled and the configured report time is reached.
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

## Third-Party Assets

- `frontend/public/assets/noto-cat-face.svg` is from Google Noto Emoji and is used for the digital employee cat avatar. See `frontend/public/assets/NotoEmoji-LICENSE.txt` for the upstream license.

## Development

### Environment

Copy `.env.example` to `.env` if you want to keep local settings in a file. The app reads these environment variables:

```env
APP_ADDR=:8080
APP_DB_PATH=data/job-hunter-agent.db
FEISHU_WEBHOOK_URL=
DISABLE_SCHEDULER=0
SOURCE_URLS=
LLM_API_KEY=
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=
```

`SOURCE_URLS` can contain comma-separated or newline-separated public recruitment URLs. Manual and scheduled crawl runs import these URLs, score them, deduplicate them, and store them in the local database.

`FEISHU_WEBHOOK_URL` is optional. Open-source users can also open the dashboard, go to Settings, paste their own Feishu incoming bot webhook URL, save it, and send a test notification. A saved dashboard webhook takes priority over the environment variable and does not require restarting the backend.

`LLM_API_KEY`, `LLM_BASE_URL`, and `LLM_MODEL` are optional. If they are not configured, the global digital employee chat uses local rule-based replies. If they are configured, the backend calls an OpenAI-compatible `/chat/completions` endpoint and falls back to local replies on failure. `OPENAI_API_KEY`, `OPENAI_BASE_URL`, and `OPENAI_MODEL` are also accepted.

Automatic Feishu duty reports require the backend process to stay running with the scheduler enabled, a Feishu webhook in Settings or `FEISHU_WEBHOOK_URL`, Automatic duty report enabled in Settings, and the configured duty report time reached in the configured time zone. The default time zone is `Asia/Shanghai`.

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

### Docker Compose

For an open-source style local deployment, Docker Compose runs the Go backend, a persistent SQLite volume, and an Nginx-served frontend that proxies `/api` to the backend.

```bash
docker compose up --build
```

Open `http://localhost:5173`. The backend is also exposed at `http://localhost:8080`.

Vercel is a good fit for the static frontend only. This project currently uses a long-running Go backend plus local SQLite and scheduled jobs, so a single Vercel deployment is not the best default. For the full product, use Docker Compose locally or deploy the backend to a service with persistent storage, then point the frontend to that backend.

### First Run Checklist

After the backend and frontend are running:

1. Open `http://localhost:5173`.
2. Go to Companies and add the recommended company pool.
3. Run Discover Sources, then validate and accept useful source candidates.
4. Go to Settings and adjust cities, directions, excluded keywords, crawl schedule, automatic source discovery, and your optional Feishu webhook.
5. Use Send Feishu Test if a webhook is configured.
6. Go back to Dashboard and run a crawl.
7. Review Opportunities, mark promising jobs as Interested or Applied, and ignore low-quality matches.
8. Refresh Daily Tasks on the Dashboard to turn the current pipeline into an actionable work queue.
9. Use the digital employee sidebar to inspect maturity, capabilities, current gaps, and the daily operating cycle.
10. Configure Automatic duty report, Duty report time, and Task SLA hours in Settings if you want stale-task tracking and scheduled reporting.
11. Go to Profile and write your candidate signals: target cities, directions, skills, preferred companies, blocked keywords, and notes.
12. Try Command Center commands such as `只看深圳 Go 后端，刷新任务`, `run crawl`, or `发送飞书日报`.
13. Mark promising jobs as Interested, then open Applications and sync application plans.
14. Open Details from an opportunity to review fit signals, application plan, risks, suggested action, notes, and decision history.
15. Use the global digital employee chat in the lower-right corner to ask what to do next or why a role fits.
16. Use Snooze, Complete, or Ignore in Daily Tasks to keep the assistant's work queue accurate.
17. Use Send to Feishu from the duty report when you want the assistant to push the current task queue and summary to your bot.
## Local Data

By default, the backend stores SQLite data under:

```text
backend/data/job-hunter-agent.db
```

Local databases, logs, build outputs, private planning docs, and environment files are ignored by Git.

## Roadmap

- Add manual URL import API and dashboard flow.
- Add the first real public-source collector.
- Improve parser adapters for more company-specific career pages and job-platform result pages.
- Improve parsing for deadline, location granularity, and application URL.
- Add richer follow-up reminders and escalation channels for daily agent tasks.
- Upgrade model chat from plain conversation to structured tool-calling planning.
- Turn interested/applied jobs into follow-up tasks with dates and application metadata.
- Add optional Feishu Base or spreadsheet sync.
- Explore resume matching and assisted application workflows after the collection pipeline is reliable.

## Contributing

This project is early and evolving. Small, focused pull requests are preferred. Please avoid adding automation that logs in to third-party platforms, bypasses anti-bot systems, or submits applications without explicit user confirmation.

## License

MIT
