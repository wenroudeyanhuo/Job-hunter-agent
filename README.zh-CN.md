# Job Hunter Agent / 秋招数字员工

![Job Hunter Agent avatar](frontend/public/assets/job-agent-avatar.png)

[English README](README.en.md) | [项目首页](README.md)

Job Hunter Agent 是一个面向秋招/校招场景的本地优先数字员工项目。它会持续发现招聘来源、采集岗位、过滤低质量信息、为岗位打分、生成每日待办，并通过网页看板和飞书机器人把“今天该处理什么”推给你。

这个项目目前优先服务技术方向求职者，默认关注深圳以及前端、后端、Java、Go、算法、AI 应用开发等岗位。它不是一个简单的爬虫集合，而是在逐步演进成一个可配置、可审计、可接入大模型和 Eino Graph 编排的求职 Agent。

## 项目状态

当前处于早期产品化阶段，已经具备可本地运行的完整闭环：

- Go 后端、React 前端、SQLite 本地持久化。
- 岗位采集、来源发现、来源验证、去重、过滤和评分。
- 候选人画像、岗位详情、投递看板、每日任务队列。
- 数字员工侧边栏、全局聊天、指令中心、建议动作审批队列。
- Agent Tool Registry，统一描述工具、风险等级、审批要求和执行预览。
- DeepSeek/OpenAI-compatible 模型可选接入，本地规则兜底。
- 多 Agent 运行时：Source Scout、Job Analyst、Memory Keeper、Planner、Tool Planner、Observer。
- 可选 Eino Graph 编排路径，默认 deterministic 编排保证零配置可运行。
- 飞书机器人测试消息、采集摘要、值班日报和 Agent Cycle 摘要。
- Docker Compose 本地部署，Qdrant 可通过 profile 启动，前端可单独静态部署。

## 架构图

```mermaid
flowchart LR
  Sources["公开招聘来源<br/>公司官网、求职平台、手动 URL"] --> Backend["Go 后端<br/>采集、解析、评分、Agent runtime"]
  Backend --> SQLite["SQLite<br/>岗位、任务、计划、Cycle、决策"]
  SQLite --> Dashboard["React 看板<br/>机会、任务、审批、记忆"]
  SQLite --> Agent["数字员工<br/>Source Scout / Job Analyst / Memory Keeper / Planner / Observer"]
  Agent --> Approval["人工审批<br/>Suggested Actions"]
  Approval --> Tools["Tool Executor<br/>采集、验证来源、刷新任务、发送报告"]
  Tools --> SQLite
  Agent -. 可选模型 .-> Model["DeepSeek / OpenAI-compatible"]
  Agent -. 可选编排 .-> Eino["Eino Graph"]
  SQLite -. 可选检索 .-> Qdrant["Qdrant 向量检索<br/>Docker Compose profile"]
  Tools -. 可选通知 .-> Feishu["飞书机器人"]
```

完整架构说明见 [docs/architecture.md](docs/architecture.md)。

## 核心能力

### 1. 招聘信息收集

- 支持手动 URL 导入和定时采集。
- 默认采集时间支持 09:00、12:00、18:00。
- 自动发现更多公司官网、社区入口、求职平台搜索入口。
- 对候选来源进行页面验证，识别招聘信号和岗位链接。
- 采集结果写入 SQLite，不只是临时显示在页面上。

### 2. 岗位过滤与评分

- 根据城市、岗位方向、公司信号、校招信号、投递链接等维度评分。
- 过滤外包、培训机构、课程销售、转正不明、无关岗位等低质量内容。
- 按申请链接和公司/标题/城市归一化结果去重。
- 在岗位详情里解释匹配原因、风险点和建议动作。

### 3. 求职工作台

- Dashboard：查看机会、任务、Agent Cycle、来源状态和建议动作。
- Profile：配置城市、方向、技能、学历、偏好公司、排除关键词。
- Applications：把感兴趣岗位转成投递计划，维护简历版本、草稿备注和跟进日期。
- Companies：维护公司池、发现来源、验证来源、接受高质量来源。
- Settings：配置采集计划、飞书机器人、模型、自动日报、任务 SLA。

### 4. 数字员工与 Agent 闭环

- 全局数字员工聊天入口，支持本地规则回复和模型增强回复。
- 聊天上下文会注入候选人画像、近期岗位、语义记忆和最近会话。
- 模型建议不会直接执行，必须进入 Suggested Actions 等待人工审批。
- 审批动作会经过统一 Agent Tool Executor 执行，并记录执行结果。
- 每次采集/日报后可以触发 Agent Cycle，记录多 Agent 观察、决策和下一步动作。
- 本地语义记忆默认使用 deterministic hash embeddings；个人部署可选启用 Qdrant 外部向量检索，DeepSeek embeddings / pgvector 保留 provider 扩展入口。

### 5. 飞书通知

- 支持在 Settings 中填写自己的飞书机器人 webhook。
- 支持发送测试消息。
- 采集完成后可发送摘要。
- 自动值班日报可包含任务队列、异常来源、推荐岗位和最新 Agent Cycle。

## 不做什么

- 不自动提交简历。
- 不自动登录第三方招聘平台。
- 不绕过验证码、滑块或反爬机制。
- 不在未确认的情况下执行外部动作。
- 不把密钥写入语义记忆。

## 快速开始

后端需要 Go 1.25 或更新版本：

```bash
cd backend
go run ./cmd/server
```

前端需要 Node.js 和 npm：

```bash
cd frontend
npm install
npm run dev
```

打开：

```text
http://localhost:5173
```

Docker Compose：

```bash
docker compose up --build
```

可选 Qdrant 语义记忆：

```powershell
$env:SEMANTIC_MEMORY_PROVIDER="qdrant"
$env:QDRANT_URL="http://qdrant:6333"
docker compose --profile qdrant up --build
```

## 环境变量

```env
APP_ADDR=:8080
APP_DB_PATH=data/job-hunter-agent.db
FEISHU_WEBHOOK_URL=
DISABLE_SCHEDULER=0
SOURCE_URLS=
AGENT_ORCHESTRATOR=deterministic
SEMANTIC_MEMORY_PROVIDER=local_hash
QDRANT_URL=
QDRANT_COLLECTION=job_hunter_memory
QDRANT_API_KEY=
LLM_PROVIDER=
LLM_API_KEY=
LLM_BASE_URL=https://api.openai.com/v1
LLM_MODEL=
DEEPSEEK_API_KEY=
DEEPSEEK_MODEL=deepseek-chat
```

常用配置：

- `SOURCE_URLS`：逗号或换行分隔的公开招聘 URL。
- `FEISHU_WEBHOOK_URL`：飞书机器人 webhook，页面 Settings 中保存的 webhook 优先级更高。
- `AGENT_ORCHESTRATOR`：默认 `deterministic`；可选 `eino_graph`。
- `SEMANTIC_MEMORY_PROVIDER`：默认 `local_hash`；设为 `qdrant` 并配置 `QDRANT_URL` 后会把语义记忆同步到 Qdrant 并优先用 Qdrant 相似检索。`deepseek_embedding`、`pgvector` 目前保留 provider 扩展入口。
- `QDRANT_URL`：可选 Qdrant 地址，例如 `http://localhost:6333`。
- `QDRANT_COLLECTION`：Qdrant collection 名称，默认 `job_hunter_memory`。
- `QDRANT_API_KEY`：可选 Qdrant API key。
- `LLM_PROVIDER`：可配置为 `deepseek` 或 OpenAI-compatible provider。
- `LLM_API_KEY` / `DEEPSEEK_API_KEY`：模型密钥，未配置时使用本地规则回复。

DeepSeek 示例：

```env
LLM_PROVIDER=deepseek
DEEPSEEK_API_KEY=your_key
DEEPSEEK_MODEL=deepseek-chat
```

## 可选 Eino Graph

```powershell
.\scripts\run-eino.ps1
```

验证：

```powershell
go test -tags eino ./internal/jobs -run TestEinoRecruitingOrchestratorRunsGraph
```

不开 `eino` build tag 时，即使配置了 `AGENT_ORCHESTRATOR=eino_graph`，项目也会安全回退到 deterministic 编排。

简历相关能力和项目证据链见 [docs/resume-proof.md](docs/resume-proof.md)。

## 第一次使用建议

1. 打开 `http://localhost:5173`。
2. 进入 Settings，设置目标城市、岗位方向、排除关键词和采集计划。
3. 如果需要飞书通知，填写自己的机器人 webhook 并发送测试消息。
4. 进入 Companies，添加推荐公司池，执行 Discover Sources。
5. 验证并接受有价值的来源。
6. 回到 Dashboard，运行一次 crawl。
7. 查看 Opportunities，标记 Interested、Applied 或 Ignore。
8. 在 Profile 中补充自己的技能、学历、偏好公司和求职备注。
9. 在 Applications 中维护投递计划、简历版本、草稿备注和跟进日期。
10. 在 Dashboard 查看 Daily Tasks、Suggested Actions 和 Agent Cycles。
11. 使用右下角数字员工聊天，询问“今天最值得投哪些岗位？”或“为什么这个岗位适合我？”。

## 常用命令

```bash
cd backend
go test ./...
go test -tags eino ./internal/jobs -run TestEinoRecruitingOrchestratorRunsGraph
```

```bash
cd frontend
npm run build
```

## 项目结构

```text
.
+-- backend
|   +-- cmd/server              # 后端入口
|   +-- internal/app            # 应用装配和调度
|   +-- internal/config         # 环境配置
|   +-- internal/crawl          # 采集 runner 和 scheduler
|   +-- internal/db             # SQLite schema 和连接
|   +-- internal/domain         # 共享领域类型
|   +-- internal/http           # REST API handlers 和 routes
|   +-- internal/jobs           # 岗位、评分、Agent runtime、语义记忆、工具注册表
|   +-- internal/notify         # 飞书 webhook 通知
+-- frontend
    +-- src                     # React 看板
```

## 本地数据

默认 SQLite 数据库路径：

```text
backend/data/job-hunter-agent.db
```

本地数据库、日志、构建产物、私有计划文档和环境变量文件已经通过 `.gitignore` 忽略。

## 部署说明

- 前端：`npm run build` 后可以作为静态资源部署。
- 后端：需要支持长时间运行的 Go 进程和持久化存储。
- Scheduler：自动采集、自动来源发现、任务 SLA、飞书日报都依赖常驻后端。
- SQLite：适合本地优先使用；如果要做公开多用户服务，建议迁移到 Postgres。
- Vercel：适合部署前端，不适合作为当前完整产品的唯一部署方式。

更多开源运行说明见 [docs/open-source-setup.md](docs/open-source-setup.md)。

v1.0 的项目终点和验收标准见 [docs/v1.0-goal.md](docs/v1.0-goal.md)。

数字员工产品化方案见 [docs/product-agent-roadmap.zh-CN.md](docs/product-agent-roadmap.zh-CN.md)。

## 路线图

- 提升更多公司官网和求职平台的岗位解析质量。
- 扩大默认公司池和来源发现策略，让数据不固定在少量头部公司。
- 将模型聊天升级为更结构化的工具调用和任务规划。
- 将 Eino Graph 从可选适配推进到更完整的多 Agent 编排。
- 引入可替换的向量数据库或外部 embedding provider。
- 增加简历版本模板、投递草稿生成和更细的跟进提醒。
- 探索 Feishu Base、表格或其他外部系统同步。

## 贡献

欢迎提交 issue 和 PR。请避免加入自动登录、绕过反爬、未经确认自动投递等能力。

## 第三方素材

- `frontend/public/assets/noto-cat-face.svg` 来自 Google Noto Emoji，用于数字员工猫头像。上游许可证见 `frontend/public/assets/NotoEmoji-LICENSE.txt`。

## License

MIT
