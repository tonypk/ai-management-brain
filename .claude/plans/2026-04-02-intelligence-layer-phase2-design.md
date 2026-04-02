# Intelligence Layer Phase 2 — 增强 Recommendations with World Model

**Date:** 2026-04-02
**Status:** Approved
**Prerequisite:** Phase 1 (Team World Model) — deployed

## 目标

让现有 Recommendation 系统消费 World Model 数据，新增 6 种 World Model 驱动的建议模式 + 3 个实时触发器。不建新表，不加新 cron，不改前端——复用现有全部基础设施。

## 约束

| 维度 | 决策 |
|---|---|
| 架构方式 | 增强现有 Recommendations，不建独立 Suggestions 实体 |
| 触发场景 | 全部 6 种（知识孤岛、协作断裂、资源配对、风险预警、成长机会、隐性问题） |
| 前端范围 | 不改（现有 Recommendation 卡片和页面已覆盖） |
| 数据库 | 零新表、零 migration |
| 成本 | DailyScan 仍然 1 次 Sonnet/天；实时触发器零 LLM 成本（模板） |

## 1. DailyScan World Model 增强

### 现状

`recommender.DailyScan()` 每天 10:30 收集 8 个数据源，用 Sonnet 生成最多 5 条 recommendation：
1. Execution signals (7d)
2. Communication events (7d)
3. Overdue tasks + blocked projects
4. Metric trends (30d)
5. Goal deviations (current cycle)
6. Employee trends (14d)
7. Pending recommendations (dedup)
8. Memory highlights (employee patterns)

### 改动

新增**第 9 个数据源** `worldModelContext`，通过 `worldmodel.Service.ForRecommenderContext()` 获取结构化 World Model 摘要。

`ForRecommenderContext(ctx, tenantID)` 返回：
```go
type RecommenderContext struct {
    KnowledgeSilos    []SiloEntry       // skill 只有 1 人掌握, confidence > 0.7
    CollabCandidates  []CollabCandidate // 做相关工作但无 relationship 记录的员工对
    SkillMatches      []SkillMatch      // blocker ↔ skill 配对候选
    EscalatingBlockers []BlockerInfo    // recurrence_count >= 3 且 active
    GrowthSignals     []GrowthSignal    // 近 7 天的 growth events
    RiskInsights      []InsightInfo     // risk/opportunity 维度 insights, confidence > 0.6
}
```

### Prompt 扩展

在 DailyScan prompt 末尾追加 World Model section：

```
## Team World Model Analysis

Based on the team's persistent knowledge graph, also check for these patterns:

1. **Knowledge Silos (bus factor=1)**: {silos_data}
   → If found, recommend knowledge sharing or cross-training

2. **Collaboration Gaps**: {collab_candidates_data}
   → If people work on related problems without collaborating, recommend pairing

3. **Skill-Blocker Matches**: {skill_matches_data}
   → If someone's blocker matches another's expertise, recommend pairing

4. **Escalating Blockers**: {escalating_blockers_data}
   → If a blocker recurs 3+ times, recommend escalation or process change

5. **Growth Opportunities**: {growth_signals_data}
   → If someone recently leveled up a skill, suggest stretch assignments

6. **Risk Patterns**: {risk_insights_data}
   → Surface team-level risks from AI insights (rhythm anomalies, etc.)

Include world_model evidence in the evidence field when generating recommendations from these patterns.
```

### Evidence 扩展

现有 evidence 结构已支持自定义字段。World Model 驱动的 recommendation 在 evidence 中增加：
```json
{
  "signals": [...],
  "employees": [...],
  "world_model_evidence": {
    "type": "knowledge_silo",
    "skill": "payment_module",
    "sole_holder": "Alice",
    "confidence": 0.85,
    "last_seen": "2026-04-01"
  }
}
```

## 2. 实时触发器

### 触发时机

每次 check-in 提交后，World Model extractor 完成提取后，调用现有的 `recommender.RealtimeEvaluate()` — 新增 event type `"world_model_extraction_complete"`。

传入 payload：
```go
type WorldModelExtractionPayload struct {
    EmployeeID    pgtype.UUID
    EmployeeName  string
    NewBlockers   []string   // 本次提取发现的新 blocker categories
    SkillChanges  []string   // 本次提取发现的 skill 变化
    SentimentScore float64   // 本次 report 的情绪分数
}
```

### 触发器 1：Blocker 恶化（blocker_escalation）

```
条件：
  - employee 有 blocker 满足 recurrence_count >= 3 且 status = 'active'
  - 最近 72h 无该 employee + category="people" 的 pending recommendation

输出：
  category: "people"
  priority: "high"
  title: "{employee} 的 {blocker_category} blocker 反复出现 ({count}次)"
  description: "该 blocker 首次出现于 {first_seen}，已反复 {count} 次未解决。建议安排会议了解根因。"
  suggested_actions:
    - { type: "flag_risk", params: { risk_description: "..." } }
    - { type: "schedule_meeting", params: { employee_name: "...", topic: "..." } }
  evidence: { world_model_evidence: { type: "blocker_escalation", ... } }
  target_entity_type: "employee"
  target_entity_id: {employee_id}
  source: "realtime_trigger"
```

### 触发器 2：技能配对（skill_match）

```
条件：
  - 本次提取发现新 blocker（NewBlockers 非空）
  - 查 world_model_skills 找到团队中另一人拥有匹配 skill，confidence > 0.6
  - 最近 72h 无该配对（两人）的 pending recommendation

输出：
  category: "people"
  priority: "medium"
  title: "建议配对：{helper} 可以帮 {blocked} 解决 {blocker_category} 问题"
  description: "{helper} 拥有 {skill}（confidence {conf}%），可以帮助 {blocked} 解决当前 {blocker_category} blocker。"
  suggested_actions:
    - { type: "send_message", params: { employee_name: "{helper}", message: "..." } }
    - { type: "send_message", params: { employee_name: "{blocked}", message: "..." } }
  evidence: { world_model_evidence: { type: "skill_match", ... } }
  source: "realtime_trigger"
```

### 触发器 3：情绪 + World Model 联合（compound_risk）

```
条件：
  - employee 最近 3 天情绪连续下滑（每天的 sentiment score 递减）
  - 且该 employee 有至少 1 个 active blocker
  - 最近 72h 无该 employee 的 pending "people" recommendation

输出：
  category: "people"
  priority: "high"
  title: "{employee} 情绪持续下滑且有未解决 blocker"
  description: "情绪趋势：{scores}。同时有 {count} 个未解决 blocker（{categories}）。建议尽快 1:1 了解情况。"
  suggested_actions:
    - { type: "schedule_meeting", params: { employee_name: "...", topic: "1:1 check-in" } }
    - { type: "send_message", params: { employee_name: "...", message: "关心问候" } }
  evidence: { world_model_evidence: { type: "compound_risk", ... } }
  target_entity_type: "employee"
  target_entity_id: {employee_id}
  source: "realtime_trigger"
```

### Dedup

所有触发器复用现有 `FindDuplicateRecommendation` 查询（基于 category + target_entity_type + target_entity_id + status=pending + 72h window）。DailyScan 和实时触发器不会重复。

## 3. 新增 sqlc 查询

加到 `sql/queries/world_model.sql`：

### FindKnowledgeSilos
```sql
-- 某 skill 在 tenant 内只有 1 人掌握且 confidence > 0.7
SELECT skill_name, employee_id, confidence, mention_count
FROM world_model_skills
WHERE tenant_id = $1
  AND confidence > 0.7
  AND skill_name IN (
    SELECT skill_name FROM world_model_skills
    WHERE tenant_id = $1 AND confidence > 0.5
    GROUP BY skill_name HAVING count(*) = 1
  )
ORDER BY confidence DESC;
```

### FindSkillMatchForBlocker
```sql
-- 某 blocker category 对应的 skill 持有者（排除 blocker 持有者自己）
SELECT s.employee_id, s.skill_name, s.confidence, s.proficiency
FROM world_model_skills s
WHERE s.tenant_id = $1
  AND s.skill_name ILIKE '%' || $2 || '%'
  AND s.confidence > 0.6
  AND s.employee_id != $3
ORDER BY s.confidence DESC
LIMIT 3;
```

### GetRecentSentimentTrend
```sql
-- 某 employee 最近 N 天的情绪分数
SELECT DATE(submitted_at) AS report_date,
       AVG(sentiment_score) AS avg_sentiment
FROM reports
WHERE tenant_id = $1
  AND employee_id = $2
  AND submitted_at > now() - ($3 || ' days')::interval
GROUP BY DATE(submitted_at)
ORDER BY report_date DESC;
```

## 4. 文件改动清单

### 修改（5 个文件）

| 文件 | 改动 |
|---|---|
| `internal/brain/recommender.go` | DailyScan: 调用 ForRecommenderContext, 扩展 prompt; RealtimeEvaluate: 新增 "world_model_extraction_complete" 分支 |
| `internal/worldmodel/service.go` | 新增 `ForRecommenderContext()` 方法，返回 `RecommenderContext` 结构 |
| `cmd/brain/main.go` | extractor 完成后调用 `recommender.RealtimeEvaluate("world_model_extraction_complete", payload)` |
| `sql/queries/world_model.sql` | 新增 3 个查询 |
| `internal/db/sqlc/world_model.sql.go` | sqlc generate 自动更新 |

### 新增（2 个文件）

| 文件 | 内容 |
|---|---|
| `internal/worldmodel/triggers.go` | 3 个触发器函数：EvaluateBlockerEscalation, EvaluateSkillMatch, EvaluateCompoundRisk |
| `internal/worldmodel/triggers_test.go` | 触发器单元测试 |

### 不改

- 数据库 schema：零 migration
- Cron jobs：不新增
- Telegram：不改（`/recs`, `/execute` 已覆盖）
- MCP：不改（`get_recommendations`, `execute_recommendation` 已覆盖）
- Frontend：不改（Recommendation 组件已覆盖）
- Dispatcher/Executor：不改（action types 足够）

## 5. 数据流

```
                     ┌─────────────────────────────────┐
                     │        DailyScan (10:30)         │
                     │  8 existing sources              │
                     │  + World Model (9th source)      │
                     │  → Sonnet prompt with 6 patterns │
                     │  → max 5 recommendations/day     │
                     └───────────────┬─────────────────┘
                                     │
                                     ▼
                          ┌──────────────────┐
                          │  recommendations  │
                          │     table         │
                          │  (existing)       │
                          └────────┬─────────┘
                                   │
              ┌────────────────────┼────────────────────┐
              ▼                    ▼                     ▼
       Telegram /recs        Frontend cards        MCP tools
       (existing)            (existing)            (existing)

  ═══════════════════════════════════════════════════════

    Check-in submitted
         │
         ▼
    World Model Extractor (Haiku)
         │
         ▼
    RealtimeEvaluate("world_model_extraction_complete")
         │
         ├─→ blocker_escalation trigger
         ├─→ skill_match trigger
         └─→ compound_risk trigger
              │
              ▼ (if condition met + no dedup)
         recommendations table → same delivery channels
```

## 6. 成本影响

| 项目 | 变化 |
|---|---|
| DailyScan Sonnet 调用 | prompt 增加 ~500 tokens（World Model 上下文），成本不变 |
| 实时触发器 | 零 LLM 成本（全模板） |
| sqlc 查询 | 轻量 SQL，可忽略 |
| **总增量** | **$0.00/天** |
