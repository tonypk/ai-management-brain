# AI Management Brain

## Skill Evolution (持续迭代指令)

当用户说 "继续优化 skill"、"skill evolution"、"迭代 skill" 时:

1. 读取 `.claude/plans/skill-evolution.md`
2. 找到最新的未完成 Phase 和 Step（第一个未勾选的 `- [ ]`）
3. 执行该 Step
4. 完成后在 plan 中勾选 `- [x]` 并更新 Progress Log
5. 告诉用户完成了什么 + 下一步是什么
6. 每完成一个 Phase，用 `clawhub publish` 发布新版本

Skill 路径: `openclaw-skill/`
