# Demo Video Script — AI Management Brain × ChatGPT

**时长目标**：2 分 30 秒
**录制工具**：Screen Studio / OBS + ChatGPT 网页版
**配乐建议**：轻快科技感 BGM，无人声（字幕替代）

---

## Scene 1: Hook — 0:00~0:15

**画面**：黑屏渐入，大字标题动画

```
当你的 AI 助手变成管理团队...
会发生什么？
```

**字幕**：`AI Management Brain — Your AI-Powered C-Suite`

切换：fade to ChatGPT 界面

---

## Scene 2: 连接 MCP — 0:15~0:30

**画面**：ChatGPT Settings 页面

**操作**：
1. 打开 Settings → MCP Servers
2. 添加 Server URL：`https://manageaibrain.com/mcp`
3. 输入 API Key
4. 显示 "Connected — 9 tools available"

**字幕**：`一个 URL，9 个管理工具，即插即用`

---

## Scene 3: 晨会快报 — 0:30~1:10

**画面**：ChatGPT 对话界面

**输入**（打字动画）：
```
帮我做今天的晨会快报：查看团队签到状态和缺勤警报，总结我需要关注的事
```

**ChatGPT 反应**（高亮 tool 调用）：
1. 🔧 `get_team_status` → 返回 12/15 已签到
2. 🔧 `get_alerts` → 返回 2 人连续 3 天未签到

**ChatGPT 输出**（逐行显示）：
```
📋 晨会快报 — 3月24日

✅ 签到率：80%（12/15）
⚠️ 关注：David 连续 3 天未签到（建议当面了解情况）
📌 今日待办：先找 David 聊聊，下午跟进 Sarah 和 Tom 的日报
```

**字幕**：`30 秒完成以前需要翻 3 个系统才能看到的信息`

---

## Scene 4: 战略决策 — 1:10~2:00

**画面**：新对话

**输入**：
```
我们要不要进日本市场？先切到稻盛和夫的管理哲学，然后开一次董事会讨论
```

**ChatGPT 反应**（高亮 tool 调用链）：
1. 🔧 `switch_mentor(inamori)` → ✅ 已切换至稻盛和夫
2. 🔧 `board_discuss("Should we expand to Japan?")` → 6 位 C-Suite 依次发言

**ChatGPT 输出**（快速滚动，停在最终总结）：
```
🏛️ 董事会决议摘要

CEO：日本市场有战略价值，但需要本地合作伙伴
CFO：前期投入约 $50K-80K，6 个月 breakeven 可行
CMO：先攻 IT/SaaS 垂直市场，避免 B2C
CTO：需要日语 i18n + 日本数据合规
CHRO：建议招 1 名日本本地 BD
COO：Q3 启动，Q4 pilot 5 家客户

📌 综合建议：推荐进入，分两阶段执行
```

**字幕**：`一个问题，6 个视角，60 秒出结果`

---

## Scene 5: Closing — 2:00~2:30

**画面**：三栏并排展示

```
┌──────────────┬──────────────┬──────────────┐
│   Claude     │   ChatGPT    │   Gemini     │
│   ✅ stdio   │   ✅ HTTP    │   ✅ HTTP    │
│   MCP native │   MCP 1.0    │   MCP 1.0    │
└──────────────┴──────────────┴──────────────┘
```

**大字**：
```
AI Management Brain
9 Tools · 16 Mentors · 6 C-Suite Seats
One MCP, Any AI Client

manageaibrain.com
```

**字幕**：`免费试用 — manageaibrain.com`

Fade out.

---

## 录制清单

- [ ] ChatGPT 账号已配置 MCP endpoint
- [ ] 团队数据已准备好（至少 10 个员工、3 天签到数据）
- [ ] 测试 3 个场景的 tool 调用全部正常
- [ ] Screen Studio 设置好 1080p + 光标放大 + 按键显示
- [ ] BGM 文件就绪
- [ ] 字幕文稿（可从本文档提取）
