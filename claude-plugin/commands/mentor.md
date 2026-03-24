---
description: Switch mentor philosophy or view current mentor
argument-hint: [mentor-id]
allowed-tools: [Read, Write]
---

# Mentor Management

Switch the active mentor philosophy or view the current one.

**Arguments:**
- No argument: show current mentor and list all available mentors
- `<mentor-id>`: switch to specified mentor

**Available mentors:**

| ID | Name | Style |
|---|---|---|
| musk | Elon Musk | First principles, urgency, 10x thinking |
| inamori | Kazuo Inamori | Altruism, team harmony, respect |
| ma | Jack Ma | Embrace change, teamwork, customer focus |
| dalio | Ray Dalio | Radical transparency, principles-driven |
| grove | Andy Grove | OKR-driven, data-focused, high-output |
| ren | Ren Zhengfei | Wolf culture, self-criticism, striver |
| son | Masayoshi Son | 300-year vision, bold bets |
| jobs | Steve Jobs | Simplicity, excellence pursuit |
| bezos | Jeff Bezos | Day 1 mentality, customer obsession |
| buffett | Warren Buffett | Long-term value, patience |
| zhangyiming | Zhang Yiming | Delayed gratification, data-driven |
| leijun | Lei Jun | Extreme value, user participation |
| caodewang | Cao Dewang | Industrial spirit, craftsmanship |
| chushijian | Chu Shijian | Ultimate focus, quality obsession |

**Process:**

1. Read `~/.boss-ai-agent/config.json`.
2. If no argument: display current mentor and the full table above.
3. If argument provided:
   - Validate it's a known mentor ID.
   - Update `config.json` with new mentor.
   - Explain what changes: check-in questions, chase style, priority focus, summary lens.
