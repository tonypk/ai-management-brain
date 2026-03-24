# ChatGPT + AI Management Brain — 场景化 Prompt

将以下 Prompt 直接粘贴到 ChatGPT 对话中即可。前提：已在 ChatGPT Settings → MCP 中配置好 `manageaibrain.com/mcp` 端点。

---

## Scenario 1: 晨会快报 — Morning Briefing

> **适用场景**：管理者每天早上 9:00 打开 ChatGPT，一句话获取团队全貌

**Prompt：**

```
我是一个管理 15 人团队的总监。请帮我做今天的晨会快报：

1. 先查看今天的团队签到状态（谁还没提交日报）
2. 再看看有没有连续缺勤的警报
3. 根据以上信息，用 3 个 bullet points 总结今天我需要关注的事项
4. 最后建议我应该先找谁聊聊

请用简洁的中文回答，像一个高效的行政助理那样汇报。
```

**预期 tool 调用链：**
`get_team_status` → `get_alerts` → ChatGPT 综合分析

---

## Scenario 2: 一对一准备 — 1:1 Meeting Prep

> **适用场景**：和下属做周度 1:1 前，快速掌握对方近况

**Prompt：**

```
我下午要和 Maria 做 1:1 meeting。请帮我准备：

1. 先拉取 Maria 的员工档案（查看她最近的签到率、情绪趋势、日报内容）
2. 再看看本周的团队报告，了解她在团队中的排名
3. 根据以上数据，帮我列出：
   - 3 个可以表扬她的点（如果有的话）
   - 2 个需要关心或跟进的问题
   - 1 个开场破冰的建议话题

请用教练式对话的语气来写，不要太正式。
```

**预期 tool 调用链：**
`get_employee_profile(Maria)` → `get_report(weekly)` → ChatGPT 生成 1:1 议程

---

## Scenario 3: 战略决策 — Strategic Board Discussion

> **适用场景**：面临重大业务决策，需要多角度分析

**Prompt：**

```
我们是一家 50 人的 SaaS 公司，目前月营收 $200K，主要市场在东南亚。
我正在考虑是否应该进入日本市场。

请帮我做一次虚拟董事会讨论：
1. 先用稻盛和夫的管理哲学（他是日本经营之圣，最适合分析日本市场）
2. 然后召开全体 C-Suite 董事会，让 CEO、CFO、CMO、CTO、CHRO、COO 各自发表意见
3. 讨论完后，再单独问 CFO：如果进入日本市场，前 12 个月需要准备多少预算？

最后请帮我总结为一份一页的决策备忘录（Decision Memo），包含：推荐/不推荐、关键理由、下一步行动、风险提示。
```

**预期 tool 调用链：**
`switch_mentor(inamori)` → `board_discuss("Should we expand to Japan market?...")` → `chat_with_seat(cfo, "Budget for Japan market entry...")` → ChatGPT 综合生成 Decision Memo

---

## 使用提示

- 这些 Prompt 设计为 **一次粘贴，自动触发多个 tool**，不需要用户手动指定调用哪个工具
- ChatGPT 会根据 tool description 自动判断调用顺序
- 每个场景都包含"数据获取 → AI 分析 → 可行动的输出"三步结构
- 建议在 ChatGPT 中使用 GPT-4o 或更高模型以获得最佳 tool 调用效果
