# Architecture

This project is a personal-first recruiting digital employee. The core design keeps local ownership of data while letting the agent use optional model, Eino Graph, Qdrant, and Feishu integrations when the user configures them.

## System View

```mermaid
flowchart LR
  subgraph External["External Inputs"]
    PublicSources["Public career sites"]
    JobPlatforms["Job platforms and search pages"]
    ManualURLs["Manual URL import"]
    UserFeedback["User decisions<br/>Interested / Applied / Ignore"]
  end

  subgraph Backend["Go Backend"]
    API["HTTP API<br/>Gin"]
    Scheduler["Scheduler<br/>09:00 / 12:00 / 18:00"]
    Crawlers["Collectors and Importers<br/>official adapters + generic HTML + JSON-LD"]
    Scoring["Filtering and Scoring<br/>profile rules + decision feedback"]
    AgentRuntime["Agent Runtime<br/>Source Scout / Job Analyst / Memory Keeper / Planner / Tool Planner / Observer"]
    ToolRegistry["Structured Agent Tool Registry<br/>schema, risk, approval, preview"]
    ApprovalQueue["Human Approval Queue<br/>Suggested Actions"]
    ToolExecutor["Tool Executor<br/>approved actions only"]
  end

  subgraph Storage["Local-first Storage"]
    SQLite["SQLite<br/>jobs, sources, tasks, plans, cycles, decisions"]
    SemanticMemory["Semantic Memory<br/>jobs, decisions, preference reflections"]
    Qdrant["Optional Qdrant<br/>external vector search / Compose profile"]
  end

  subgraph OptionalAI["Optional AI Orchestration"]
    DeepSeek["DeepSeek or OpenAI-compatible chat"]
    Eino["Optional Eino Graph<br/>-tags eino"]
  end

  subgraph Outputs["User Surfaces"]
    Dashboard["React Dashboard"]
    Chat["Digital employee chat"]
    Feishu["Feishu bot reports"]
    Docs["GitHub docs"]
  end

  PublicSources --> Crawlers
  JobPlatforms --> Crawlers
  ManualURLs --> API
  Scheduler --> Crawlers
  API --> Crawlers
  Crawlers --> Scoring
  UserFeedback --> Scoring
  Scoring --> SQLite
  SQLite --> SemanticMemory
  SemanticMemory -. optional sync/search .-> Qdrant
  SQLite --> AgentRuntime
  SemanticMemory --> AgentRuntime
  DeepSeek -. model-enhanced decisions .-> AgentRuntime
  Eino -. graph runner .-> AgentRuntime
  AgentRuntime --> ToolRegistry
  ToolRegistry --> ApprovalQueue
  ApprovalQueue --> ToolExecutor
  ToolExecutor --> SQLite
  ToolExecutor --> Feishu
  ToolExecutor --> AgentRuntime
  API --> Dashboard
  API --> Chat
  SQLite --> Dashboard
  AgentRuntime --> Feishu
  Docs --> Dashboard
```

## Agent Loop

```mermaid
flowchart TD
  Observe["Observe<br/>jobs, sources, tasks, plans, memory"]
  SourceScout["Source Scout<br/>source health and expansion"]
  JobAnalyst["Job Analyst<br/>quality, parser gaps, strong matches"]
  MemoryKeeper["Memory Keeper<br/>semantic coverage and history"]
  Planner["Planner<br/>daily work and safe next steps"]
  ToolPlanner["Tool Planner<br/>structured tool calls"]
  Approval["Human approval gate<br/>Suggested Actions"]
  Executor["Tool Executor<br/>crawl, inspect sources, generate plans, refresh tasks, send report, validate sources"]
  Observer["Observer<br/>execution receipt and re-plan signal"]
  Replan["Re-plan Proposal<br/>next approval-gated actions"]
  Persist["Persist<br/>Agent Cycle + autonomy_plan"]

  Observe --> SourceScout --> JobAnalyst --> MemoryKeeper --> Planner --> ToolPlanner
  ToolPlanner --> Approval
  Approval -->|approved| Executor
  Approval -->|dismissed| Persist
  Executor --> Observer --> Replan --> Persist
  Persist --> Observe
```

## Data Quality Pipeline

```mermaid
flowchart LR
  SourcePool["Source pool<br/>official + discovered candidates"]
  Validate["Validate source<br/>reachable, signals, job cards"]
  Crawl["Crawl pages<br/>retry and source health"]
  Parse["Parse jobs<br/>official APIs, generic HTML, JSON-LD"]
  Normalize["Normalize<br/>company, title, city, apply URL"]
  Score["Score<br/>profile fit + feedback learning"]
  Store["Store<br/>SQLite jobs and semantic memory"]
  Reflect["Reflect<br/>learned preferences from decisions"]
  Review["Review queue<br/>manual_check and parser gaps"]

  SourcePool --> Validate --> Crawl --> Parse --> Normalize --> Score --> Store
  Store --> Reflect --> Store
  Parse -->|low confidence| Review
  Review -->|accepted signals| Score
  Review -->|parser gap| SourcePool
```

## Personal Deployment Boundary

```mermaid
flowchart TB
  Local["Required local stack<br/>Go backend + React frontend + SQLite"]
  Optional["Optional integrations"]
  Model["DeepSeek / OpenAI-compatible model"]
  Compose["Docker Compose<br/>backend + frontend + volumes"]
  Vector["Qdrant vector search<br/>profile: qdrant"]
  Notify["Feishu incoming bot"]
  Graph["Eino Graph build tag"]

  Local --> Compose
  Compose --> Optional
  Optional --> Model
  Optional --> Vector
  Optional --> Notify
  Optional --> Graph
```

## Key Boundaries

- Local SQLite remains the source of truth for personal use.
- External integrations are opt-in and can fail without blocking local operation.
- Model output never executes directly; it becomes an approval-gated action.
- No automatic resume submission, third-party login automation, captcha bypass, or secret storage in semantic memory.
