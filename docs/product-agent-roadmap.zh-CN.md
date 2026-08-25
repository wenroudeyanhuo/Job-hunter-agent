# 数字员工产品化路线图

这份路线图用于把 Job Hunter Agent 从“求职工具”推进到“可持续工作的多 Agent 数字员工”。

## 1. Agent Tool Registry

目标：让 Agent 的动作不是散落按钮，而是统一、可审计、可审批、可执行的工具调用。

当前进展：

- 已新增 Agent Tool Registry。
- 已为 `run_crawl`、`discover_sources`、`refresh_tasks`、`sync_application_plans`、`send_feishu_report`、`add_recommended_and_crawl`、`review_strong_matches`、`review_manual_check` 注册工具元数据。
- 每个工具包含描述、输入 schema、风险等级、审批要求和执行预览。
- Suggested Actions 会展示工具风险和执行预览。
- 审批后的动作通过统一 Agent Tool Executor 执行。

后续：

- 为工具增加结构化参数解析。
- 为工具执行增加重试策略和超时配置。
- 为高风险工具增加二次确认。
- 将工具列表暴露给 Eino/模型 planner，作为可调用能力清单。

## 2. 多 Agent 闭环

目标：从“顺序报告”升级为“计划 -> 工具调用 -> 观察结果 -> 再计划”。

当前进展：

- 已有 Source Scout、Job Analyst、Memory Keeper、Planner。
- 已新增 Observer 角色，用于总结工具执行结果。
- 已有 Agent Cycle 持久化。
- Eino Graph 已升级为 `SourceScout -> JobAnalyst -> MemoryKeeper -> Planner -> ToolPlanner -> Observer -> Finalize`。
- Agent Cycle 会持久化 `autonomy_plan`，包含待审批工具、风险等级、执行后观察和 re-plan 信号。
- 已有审批队列和工具执行入口。
- 工具审批成功后会记录 `tool_observer` Agent Cycle，为下一轮计划提供观察依据。

后续：

- Planner 输出结构化 tool calls。
- Tool Executor 产出 execution receipts。
- Tool Executor 仍然通过人工审批执行，不做无授权自动投递。
- Observer 汇总 execution receipts，并触发下一轮 Agent Cycle。
- Agent Cycle 继续扩展记录每轮输入、工具结果、失败原因和后续建议。

## 3. 数据质量

目标：岗位数据要足够准确，不能只是“采到网页标题”。

当前进展：

- 已有推荐来源池、来源发现、来源验证。
- 已有部分公司官方适配器。
- 已有通用 HTML link/card 发现。
- 已有 landing page cleanup。
- 已新增 source quality score，用于衡量来源池健康度。
- 已新增 parser gap 任务，无法解析的招聘落地页会进入 Daily Tasks。
- 公共 URL 导入遇到临时 5xx/网络失败会进行轻量重试。
- 通用 HTML 解析已支持非链接结构化岗位卡片、分页入口、城市、方向标签和截止时间。

后续：

- 增加更多公司官方 parser。
- 加强通用列表页解析：分页、职位卡片、城市、截止时间、投递链接。
- 建立 parser gap 任务：无法解析的高价值来源进入 Daily Tasks。
- 根据用户 Interested/Ignore/Applied 反馈动态调整评分。
- 为 source 增加长期质量分：成功率、有效岗位数、重复率、人工检查率。

## 4. 记忆系统

目标：让数字员工记住“用户为什么选择/忽略岗位”，而不是只做关键词搜索。

当前进展：

- 已有本地 semantic memory。
- 默认 provider 是 `local_hash`。
- 已有语义搜索 API 和聊天上下文注入。
- 已新增 profile memory 和 decision memory。
- 候选人画像保存、岗位决策记录会自动写入语义记忆。
- 已新增 `SEMANTIC_MEMORY_PROVIDER` 配置入口。
- Qdrant 已接入为可选外部向量库：本地 SQLite 保留副本，配置 `SEMANTIC_MEMORY_PROVIDER=qdrant` 和 `QDRANT_URL` 后会同步向量并优先使用 Qdrant 检索。

后续：

- 抽象更完整的 MemoryProvider。
- 支持 DeepSeek embeddings、pgvector。
- 记忆分层：岗位记忆、决策记忆、偏好记忆、对话记忆。
- 每条记忆保留来源、时间、置信度和可解释引用。
- 让推荐解释能引用历史决策。

## 5. 开源用户 Onboarding

目标：别人 clone 后 10 分钟内知道怎么跑、怎么配置、怎么看到真实效果。

当前进展：

- README 已拆成中英双入口。
- 已有中文 README 和英文 README。
- 已有 Docker Compose 和开源设置文档。
- Settings 中已有飞书、模型、自动化相关配置。
- 已新增 onboarding health API 和 Settings 健康卡片，提示来源池、画像、采集记录、飞书和模型状态。
- Settings 已提供首次使用向导步骤，显示来源、画像、首次采集、飞书、模型的完成情况。

后续：

- 应用内首次启动 wizard。
- 示例数据和示例来源导入。
- 设置健康检查：数据库、scheduler、飞书、模型、来源池。
- Dashboard 增加“下一步建议”卡片。
- 文档增加真实使用路径和部署范式。

## 6. 生产可靠性

目标：从个人本地工具走向可长期运行的服务。

当前进展：

- Docker Compose 可运行完整本地产品。
- SQLite schema migration 已自动化。
- 已有 scheduler、任务 SLA、飞书日报、source health。
- 工具执行会记录 execution receipt、Agent Event 和 Observer Cycle。
- 公共 URL 采集已有轻量 retry。

后续：

- crawl retry 和 structured failure reason。
- 工具执行日志和 scheduler 日志面板。
- SQLite 备份/恢复按钮或命令。
- Postgres migration path。
- 支持 Render/Fly.io/Railway/VPS 等后端部署方式。
- 增加健康检查 API：scheduler、database、model、Feishu、source pool。

## 推荐推进顺序

1. Agent Tool Registry 和 Tool Executor。
2. 数据质量：parser gap、source quality score、更多官方 parser。
3. Observer 节点和多 Agent 闭环。
4. Onboarding wizard 和健康检查。
5. 可替换向量记忆。
6. 生产部署可靠性。
