---
name: boss-ai-agent
version: "1.0.0"
description: "Boss AI Agent — your AI management middleware. Connects boss to all systems (Telegram/Slack/GitHub/Notion/Email), 14 mentor philosophies, 9 culture packs, 7 automated scenarios. OpenClaw native-first, zero external dependency."
user-invocable: true
emoji: "🤖"
homepage: "https://manageaibrain.com"
metadata:
  openclaw:
    primaryEnv: "BOSS_AI_AGENT_API_KEY"
    requires:
      env:
        - "BOSS_AI_AGENT_API_KEY"
      config:
        - "~/.openclaw/skills/boss-ai-agent/config.json"
---

# Boss AI Agent

Your AI management middleware — connects you to all systems through mentor wisdom.

## Features

- **7 automated scenarios**: daily check-in cycle, project health patrol, smart briefing, 1:1 meeting prep, signal scanning, knowledge base, emergency response
- **14 mentor philosophies**: Musk, Inamori, Jack Ma, Dalio, Grove, Ren Zhengfei, Son, Jobs, Bezos, Buffett, Zhang Yiming, Lei Jun, Cao Dewang, Chu Shijian
- **9 culture packs**: default, Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India
- **23+ messaging platforms**: works with any channel connected to OpenClaw
- **Zero external dependency**: fully functional without any cloud account

## Install

```
clawhub install boss-ai-agent
```

## Quick Start

```
/boss-ai-agent
> How many people do you manage? 5
> What communication tools does your team use? Telegram and GitHub
> Done! Musk mode activated. First check-in at 9 AM tomorrow.
```

## Mentors

| ID | Name | Style |
|----|------|-------|
| musk | Elon Musk | First principles, urgency, 10x thinking |
| inamori | Kazuo Inamori (稻盛和夫) | Altruism, respect, team harmony |
| ma | Jack Ma (马云) | Embrace change, teamwork, customer-first |
| dalio | Ray Dalio | Radical transparency, principles-driven |
| grove | Andy Grove | OKR-driven, data-focused, high output |
| ren | Ren Zhengfei (任正非) | Wolf culture, self-criticism, striver-oriented |
| son | Masayoshi Son (孙正义) | 300-year vision, bold bets |
| jobs | Steve Jobs | Simplicity, excellence pursuit |
| bezos | Jeff Bezos | Day 1 mentality, customer obsession |
| buffett | Warren Buffett | Long-term value, patience |
| zhangyiming | Zhang Yiming (张一鸣) | Delayed gratification, data-driven |
| leijun | Lei Jun (雷军) | Extreme value, user participation |
| caodewang | Cao Dewang (曹德旺) | Industrial spirit, craftsmanship |
| chushijian | Chu Shijian (褚时健) | Ultimate focus, resilience |

## Scenarios

1. **Daily Management Cycle** — AI sends check-ins, chases non-responders, generates end-of-day summary
2. **Project Health Patrol** — scans GitHub/Linear/Jira for stale PRs, failed CI, overdue tasks
3. **Smart Daily Briefing** — cross-channel morning briefing sorted by mentor priority
4. **1:1 Meeting Assistant** — auto-generates prep document with employee data and suggested topics
5. **Periodic Signal Scanning** — monitors team channels every 30 min for urgent/warning signals
6. **Knowledge Base Management** — records decisions and notes to Notion, Google Sheets, or local files
7. **Emergency Response** — detects critical signals, alerts boss immediately, gathers rapid intel

## Cloud Platform (Optional)

Connect to manageaibrain.com for additional features:

- Web dashboard and analytics
- Full mentor configs for all 14 mentors
- Cross-team benchmarking

Set `BOSS_AI_AGENT_API_KEY` to enable. All 7 scenarios work without it.

## Migrating from management-brain?

Boss AI Agent is a new skill, not an upgrade. Your existing management-brain data is untouched.

To switch:
1. Install boss-ai-agent
2. Run onboarding (`/boss-ai-agent`)
3. Add your team members

Legacy env var `MANAGEMENT_BRAIN_API_KEY` is accepted as fallback.

## 中文说明

Boss AI Agent 是老板的 AI 管理中间件。通过已有的 OpenClaw 频道（Telegram/Slack/飞书）管理团队，零外部依赖。

**核心功能：**
- 7 大自动化场景（签到、巡检、早报、1:1、信号扫描、知识库、紧急响应）
- 14 位导师哲学（马斯克、稻盛和夫、马云等）
- 9 套文化包（适配菲律宾、新加坡、中国、美国、印度等文化差异）

安装：`clawhub install boss-ai-agent`

## Links

- Website: https://manageaibrain.com
- GitHub: https://github.com/tonypk/ai-management-brain
- ClawHub: https://clawhub.ai/tonypk/boss-ai-agent
