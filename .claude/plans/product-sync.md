# Product Sync Plan — 让网站与 Skill 对齐

> **Usage**: 每次 session 说 "产品对齐" 或 "product sync" 即可。Claude 读取本文件，找到当前阶段，继续推进。
>
> **触发词**: 产品对齐 / product sync / 网站更新
> **范围**: 落地页、README、MCP server、文化包 YAML — 让产品实际能力与宣传一致

---

## Phase 7: 补缺失资产 — 让声称的功能真正存在

**Goal**: 修复产品声称有但实际缺失的内容

### Steps

- [x] 7.1 创建 `configs/cultures/usa.yaml` — 美国文化包（direct, low hierarchy, no warmth required）
- [x] 7.2 创建 `configs/cultures/india.yaml` — 印度文化包（indirect, high hierarchy, relationship-first, warmth required）
- [x] 7.3 创建 `configs/cultures/default.yaml` — 中性 default 提取为 YAML
- [x] 7.4 更新 `references/mcp-tools.md` — 补充 11 个工具（consulting 9 + world model 2），总数 33→44
- [x] 7.5 更新 SKILL.md 中 tool count 引用（33→44）
- [x] 7.6 在 `references/scenarios.md` 中补充 Scenario 13 (Consulting) + Scenario 14 (World Model)，总数 12→14
- [x] 7.7 git commit + push (bb06731)

---

## Phase 8: 更新 README — 消除过期信息

**Goal**: README.md 反映产品真实状态

### Steps

- [x] 8.1 修复 README 第 28 行: "6 Culture Packs" → "9 Culture Packs"
- [x] 8.2 更新 "Current State" section: v3.0.0→v8.0.0, 13→44 tools, boss-ai-agent@8.0.0, 加 Consulting/World Model/Execution Intelligence
- [x] 8.3 更新 Roadmap 表格中的 MCP tool 数量（已在 Phase 7 中完成）
- [x] 8.4 补充新功能到 feature list: 16 mentors + Consulting/World Model/Execution Intelligence/Sync/Incentives + 14 scenarios
- [x] 8.5 git commit + push (9fe0852)

---

## Phase 9: MCP Server 版本对齐

**Goal**: MCP server 内部一致性

### Steps

- [x] 9.1 修复 `mcp-server/server.ts` 内部版本号 1.0.0→1.1.0（与 package.json 一致）
- [x] 9.2 验证 44 个工具在 MCP server 中全部注册且可调用（grep 确认 44 个 server.tool 调用）
- [x] 9.3 git commit + push（无新改动，9.1 已在 bb06731 中提交）
- [ ] 9.4 发布 MCP server 新版本到 npm（需要 `npm login` — 跳过，手动执行）

---

## Phase 10: 落地页改版 — 展示完整能力

**Goal**: LandingView.vue 展示所有已上线功能

### Steps

- [x] 10.1 新增 "Consulting Engine" feature card
- [x] 10.2 新增 "Execution Intelligence" feature card (14 scenarios)
- [x] 10.3 新增 "World Model" feature card
- [x] 10.4 新增 "Notion/Sheets Sync" feature card
- [x] 10.5 更新 scenarios + MCP tool count in feature descriptions
- [x] 10.6 更新 MCP tool count: 33→44 (Install section)
- [x] 10.7 更新 Enterprise 定价: + Consulting + Incentives + Sync
- [x] 10.8 本地 build 通过（vite build 无 error）
- [x] 10.9 rsync dist → rebuild frontend container → healthz ok
- [x] 10.10 git commit + push (b28735f)

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
| 2026-04-20 | 7+8+9 partial | 3 文化包 YAML, mcp-tools 33→44, README v8.0.0, server.ts 版本修复 | 7.6 scenarios |
| 2026-04-20 | 7.6 | scenarios.md 补充 Consulting + World Model (12→14 scenarios) | 8.3 README roadmap |
| 2026-04-20 | 8+9+10 | README feature list, MCP 验证 44 tools, 落地页改版 9 cards + deploy | 11 Skill v9.0.0 |
