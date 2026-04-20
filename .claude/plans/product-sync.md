# Product Sync Plan — 让网站与 Skill 对齐

> **Usage**: 每次 session 说 "产品对齐" 或 "product sync" 即可。Claude 读取本文件，找到当前阶段，继续推进。
>
> **触发词**: 产品对齐 / product sync / 网站更新
> **范围**: 落地页、README、MCP server、文化包 YAML — 让产品实际能力与宣传一致

---

## Phase 7: 补缺失资产 — 让声称的功能真正存在

**Goal**: 修复产品声称有但实际缺失的内容

### Steps

- [ ] 7.1 创建 `configs/cultures/usa.yaml` — 美国文化包（direct communication, individual accountability, data-driven）
- [ ] 7.2 创建 `configs/cultures/india.yaml` — 印度文化包（hierarchical respect, relationship-first, indirect feedback）
- [ ] 7.3 创建 `configs/cultures/default.yaml` — 把 Go 代码中 hardcoded 的 default 提取为 YAML，保持一致性
- [ ] 7.4 更新 `references/mcp-tools.md` — 补充 11 个未文档化工具（consulting 9 + world model 2），总数 33→44
- [ ] 7.5 更新 SKILL.md 中 tool count 引用（33→44）
- [ ] 7.6 在 `references/scenarios.md` 中补充 Consulting 场景的完整 step-by-step（用 9 个 consulting 工具）
- [ ] 7.7 git commit + push

---

## Phase 8: 更新 README — 消除过期信息

**Goal**: README.md 反映产品真实状态

### Steps

- [ ] 8.1 修复 README 第 28 行: "6 Culture Packs" → "9 Culture Packs"
- [ ] 8.2 更新 "Current State" section: v3.0.0→当前版本, 13 MCP tools→44, boss-ai-agent@3.0.0→8.0.0
- [ ] 8.3 更新 Roadmap 表格中的 MCP tool 数量
- [ ] 8.4 补充新功能到 feature list: Consulting Engine, World Model, Execution Intelligence, Sync, Incentives
- [ ] 8.5 git commit + push

---

## Phase 9: MCP Server 版本对齐

**Goal**: MCP server 内部一致性

### Steps

- [ ] 9.1 修复 `mcp-server/server.ts` 内部版本号 1.0.0→与 package.json 一致
- [ ] 9.2 验证 44 个工具在 MCP server 中全部注册且可调用
- [ ] 9.3 git commit + push
- [ ] 9.4 发布 MCP server 新版本到 npm（`npm publish`）

---

## Phase 10: 落地页改版 — 展示完整能力

**Goal**: LandingView.vue 展示所有已上线功能

### Steps

- [ ] 10.1 新增 "Consulting Engine" feature card — AI 麦肯锡, 结构化诊断, 行动计划, 进度追踪
- [ ] 10.2 新增 "Execution Intelligence" feature card — 风险信号, KPI 仪表盘, working memory, 逾期任务追踪
- [ ] 10.3 新增 "World Model" feature card — 团队技能图谱, 成长轨迹, 协作关系, AI 洞察
- [ ] 10.4 新增 "Data Sync" feature card — Notion/Sheets 双向同步, 冲突解决
- [ ] 10.5 更新 "7 automated scenarios" → "12 automated scenarios"
- [ ] 10.6 更新 MCP tool count: 33→44
- [ ] 10.7 更新定价层级 — Enterprise 加 Consulting + Incentives + Custom Culture Packs
- [ ] 10.8 本地预览确认无 UI 问题
- [ ] 10.9 build + deploy frontend 到服务器
- [ ] 10.10 git commit + push

---

## Phase 11: Skill v9.0.0 发布

**Goal**: skill 文档与产品完全同步后发布新版

### Steps

- [ ] 11.1 更新 SKILL.md + README.md 版本号 → 9.0.0
- [ ] 11.2 `clawhub publish openclaw-skill --slug boss-ai-agent --name "Boss AI Agent" --version 9.0.0`
- [ ] 11.3 更新 skill-evolution.md Progress Log

---

## Progress Log

| Date | Phase | What was done | Next |
|------|-------|---------------|------|
| 2026-04-20 | Gap Analysis | 完成 skill vs 网站差异分析: 3 HIGH, 7 MEDIUM, 4 LOW | Phase 7.1 |
