<script setup lang="ts">
import { NButton, NSpace, NTag, NGrid, NGi, NCard } from 'naive-ui'
</script>

<template>
  <div style="max-width: 1200px; margin: 0 auto; padding: 0 24px">
    <!-- Hero -->
    <section style="text-align: center; padding: 80px 0 40px">
      <h1 style="font-size: 48px; font-weight: 800; margin-bottom: 16px; color: #1a1a2e">AI Management Brain</h1>
      <p style="font-size: 20px; color: #64748b; max-width: 700px; margin: 0 auto 32px; line-height: 1.6">
        Your AI-powered management operating system. Assemble a virtual C-Suite, choose from 16 mentor philosophies, and manage across cultures — in Telegram, Slack, or your AI coding tools.
      </p>
      <NSpace justify="center" :size="16">
        <router-link to="/login">
          <NButton type="primary" size="large">Get Started Free</NButton>
        </router-link>
        <a href="https://github.com/tonypk/ai-management-brain" target="_blank">
          <NButton size="large">GitHub</NButton>
        </a>
      </NSpace>
      <NSpace justify="center" :size="8" style="margin-top: 24px">
        <NTag round>Open Source</NTag>
        <NTag round>ClawHub Skill</NTag>
        <NTag round>Claude Cowork Plugin</NTag>
      </NSpace>
    </section>

    <!-- Integrations -->
    <section style="padding: 40px 0 60px">
      <h2 style="text-align: center; font-size: 28px; font-weight: 700; margin-bottom: 32px">Works With Your Tools</h2>
      <NGrid :cols="5" :x-gap="20" :y-gap="20" responsive="screen" :item-responsive="true">
        <NGi v-for="item in [
          { title: 'MCP CLI', desc: 'Run MCP server locally via stdio. One API key, zero HTTP auth. Works with any MCP client.', code: 'npx -y @tonykk/management-brain-mcp' },
          { title: 'ClawHub', desc: 'Install as an OpenClaw skill. One command, zero config.', code: 'clawhub install boss-ai-agent' },
          { title: 'Claude Code', desc: 'Add as a Claude Code plugin. Your AI mentor lives inside your IDE.', code: '/boss-ai-agent' },
          { title: 'Telegram Bot', desc: 'Full-featured bot for team check-ins, reports, and board discussions.', code: '@aimanagerbrainbot' },
          { title: 'Web Dashboard', desc: 'Full management dashboard with analytics and AI C-Suite controls.', code: 'manageaibrain.com' },
        ]" :key="item.title" span="5 m:1">
          <NCard size="small" style="text-align: center; height: 100%">
            <h4 style="font-size: 16px; margin-bottom: 6px">{{ item.title }}</h4>
            <p style="font-size: 13px; color: #64748b; line-height: 1.5; margin-bottom: 12px">{{ item.desc }}</p>
            <code style="display: inline-block; background: #1e293b; color: #a5f3fc; padding: 6px 14px; border-radius: 6px; font-size: 12px">{{ item.code }}</code>
          </NCard>
        </NGi>
      </NGrid>
    </section>

    <!-- Features -->
    <section style="padding: 60px 0">
      <h2 style="text-align: center; font-size: 32px; font-weight: 700; margin-bottom: 40px">Core Features</h2>
      <NGrid :cols="3" :x-gap="24" :y-gap="24" responsive="screen" :item-responsive="true">
        <NGi v-for="f in [
          { title: 'AI C-Suite', desc: 'Assemble your virtual management team — CEO, CFO, CMO, CTO, CHRO, COO. Run board discussions where every seat weighs in on strategic questions.' },
          { title: '16 Management Mentors', desc: 'Inamori, Dalio, Grove, Ren, Son, Jobs, Bezos, Ma, Musk, plus domain experts. Each with unique management philosophies.' },
          { title: 'Consulting Engine', desc: 'Like hiring McKinsey — describe a business problem, get structured diagnosis, root cause analysis, action plans, and progress tracking across sessions.' },
          { title: 'Execution Intelligence', desc: 'AI-powered risk signals, KPI dashboards, working memory, and overdue task tracking. 14 automated scenarios keep your team on track.' },
          { title: 'World Model', desc: 'Team skills map, growth trajectories, collaboration patterns, and AI insights — all extracted automatically from daily check-ins.' },
          { title: '44 MCP Tools', desc: 'Full management API via CLI (stdio) or HTTP. Check-ins, consulting, sync, incentives, risk signals — all available to any MCP client.' },
          { title: '9 Culture Packs', desc: 'Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India — culturally adapted communication for global teams.' },
          { title: 'Notion/Sheets Sync', desc: 'Bidirectional sync with Notion and Google Sheets. Auto-detect changes, resolve conflicts, keep everything in sync.' },
          { title: 'Mentor Blending', desc: 'Combine two mentors (e.g., 70% Inamori + 30% Dalio) for a custom management philosophy.' },
        ]" :key="f.title" span="3 m:1">
          <NCard>
            <h3 style="font-size: 20px; margin-bottom: 8px">{{ f.title }}</h3>
            <p style="color: #64748b; line-height: 1.6">{{ f.desc }}</p>
          </NCard>
        </NGi>
      </NGrid>
    </section>

    <!-- C-Suite -->
    <section style="padding: 60px 0">
      <h2 style="text-align: center; font-size: 32px; font-weight: 700; margin-bottom: 8px">Your Virtual Board of Directors</h2>
      <p style="text-align: center; color: #64748b; max-width: 600px; margin: 0 auto 36px; line-height: 1.6">
        Assign the right expert to each seat. Run board discussions on any topic. Get synthesized strategic advice from multiple perspectives.
      </p>
      <NGrid :cols="6" :x-gap="16" :y-gap="16" responsive="screen" :item-responsive="true">
        <NGi v-for="seat in [
          { type: 'CEO', desc: 'Strategic vision and execution' },
          { type: 'CFO', desc: 'Financial strategy and risk' },
          { type: 'CMO', desc: 'Marketing and positioning' },
          { type: 'CTO', desc: 'Technology and architecture' },
          { type: 'CHRO', desc: 'People and culture' },
          { type: 'COO', desc: 'Operations and processes' },
        ]" :key="seat.type" span="6 m:3 l:1">
          <NCard style="background: #1e293b; color: white; text-align: center">
            <div style="font-size: 24px; font-weight: 700; margin-bottom: 4px">{{ seat.type }}</div>
            <div style="font-size: 12px; color: #94a3b8">{{ seat.desc }}</div>
          </NCard>
        </NGi>
      </NGrid>

      <NCard style="max-width: 720px; margin: 40px auto 0; background: #f8fafc">
        <div style="font-size: 12px; font-weight: 600; color: #6366f1; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 16px">
          Example: Board Discussion
        </div>
        <div style="background: #1e293b; color: #a5f3fc; padding: 10px 16px; border-radius: 8px; font-family: monospace; font-size: 14px; margin-bottom: 16px">
          /board Should we expand to the Japan market?
        </div>
        <div style="display: flex; flex-direction: column; gap: 10px">
          <div v-for="resp in [
            { role: 'CEO (Musk)', text: 'First principles — what\'s the TAM and can we 10x there?' },
            { role: 'CFO (Dalio)', text: 'Risk-adjusted ROI needs modeling. What\'s the capital requirement?' },
            { role: 'CMO (Trout)', text: 'Position as the first AI management tool for Japanese enterprise...' },
            { role: 'CHRO (Meyer)', text: 'Japanese high-context culture requires relationship-first approach...' },
          ]" :key="resp.role"
            style="font-size: 14px; color: #475569; line-height: 1.5; padding: 8px 12px; background: white; border-radius: 8px; border: 1px solid #e2e8f0"
          >
            <strong>{{ resp.role }}:</strong> {{ resp.text }}
          </div>
          <div style="font-size: 14px; color: #475569; line-height: 1.5; padding: 8px 12px; background: #f0fdf4; border-radius: 8px; border: 1px solid #86efac">
            <strong>Synthesis:</strong> Expand — but hire a local GM first. Budget $200K for 6-month pilot.
          </div>
        </div>
      </NCard>
    </section>

    <!-- Install -->
    <section style="padding: 60px 0">
      <h2 style="text-align: center; font-size: 32px; font-weight: 700; margin-bottom: 36px">Install in Seconds</h2>
      <NGrid :cols="3" :x-gap="24" :y-gap="24" responsive="screen" :item-responsive="true" style="max-width: 1100px; margin: 0 auto">
        <NGi v-for="card in [
          { title: 'MCP CLI (stdio)', desc: 'Run 44 MCP tools locally. One API key, no HTTP auth. Recommended for Claude Code, Hermes Agent, and any stdio-capable client.', cmd: 'npx -y @tonykk/management-brain-mcp', features: ['Only needs MANAGEMENT_BRAIN_API_KEY', 'Lower latency — no HTTP overhead', 'Works with any MCP stdio client'] },
          { title: 'ClawHub (OpenClaw Skill)', desc: 'Full management middleware as an OpenClaw skill. 16 mentors, 9 cultures, 44 MCP tools.', cmd: 'clawhub install boss-ai-agent', features: ['Advisor Mode works offline', 'Team Ops via CLI or HTTP MCP', 'Auto cron jobs for check-ins'] },
          { title: 'MCP HTTP', desc: 'Cloud-hosted MCP endpoint for web-based clients. ChatGPT, Gemini, and browser-based MCP tools.', cmd: 'manageaibrain.com/mcp', features: ['Streamable HTTP transport', 'Bearer token auth', 'No local install needed'] },
        ]" :key="card.title" span="3 m:1">
          <NCard>
            <h3 style="font-size: 20px; margin-bottom: 8px">{{ card.title }}</h3>
            <p style="color: #64748b; font-size: 14px; line-height: 1.5; margin-bottom: 16px">{{ card.desc }}</p>
            <div style="background: #1e293b; padding: 12px 16px; border-radius: 8px; margin-bottom: 16px">
              <code style="color: #a5f3fc; font-size: 14px">{{ card.cmd }}</code>
            </div>
            <div v-for="f in card.features" :key="f" style="padding: 6px 0; color: #475569; font-size: 14px; border-bottom: 1px solid #f1f5f9">
              ✓ {{ f }}
            </div>
          </NCard>
        </NGi>
      </NGrid>
    </section>

    <!-- Pricing -->
    <section style="padding: 60px 0">
      <h2 style="text-align: center; font-size: 32px; font-weight: 700; margin-bottom: 40px">Simple Pricing</h2>
      <NGrid :cols="3" :x-gap="24" :y-gap="24" responsive="screen" :item-responsive="true" style="max-width: 960px; margin: 0 auto">
        <NGi v-for="plan in [
          { name: 'Open Source', price: 'Free', desc: 'Self-hosted, Apache 2.0', features: ['Complete Bot + API + Dashboard', '16 mentors + 9 culture packs', 'AI C-Suite with board discussions', 'Single tenant, up to 10 employees', 'ClawHub skill included', 'Docker one-click deploy'], cta: 'Deploy Now', link: 'https://github.com/tonypk/ai-management-brain', featured: false },
          { name: 'Cloud Pro', price: '$29/mo', desc: 'Up to 20 people', features: ['Cloud hosted, zero maintenance', 'All mentors + custom creation', 'Full C-Suite + board discussions', 'Multi-channel (Telegram + Slack)', 'Mentor blending', 'Analytics dashboard'], cta: 'Start Free Trial', link: '/login', featured: true },
          { name: 'Enterprise', price: '$299/mo', desc: 'Up to 100 people', features: ['Everything in Pro', 'AI Consulting Engine', 'Incentive Calculation', 'Notion/Sheets bidirectional sync', 'Multi-tenant (departments)', 'BYOK (your own API key)', 'SSO + RBAC', 'Custom mentors and culture packs'], cta: 'Contact Us', link: '/login', featured: false },
        ]" :key="plan.name" span="3 m:1">
          <NCard :style="plan.featured ? 'border: 2px solid #6366f1; box-shadow: 0 4px 24px rgba(99,102,241,0.15)' : ''">
            <NTag v-if="plan.featured" type="primary" size="small" style="margin-bottom: 12px">Most Popular</NTag>
            <h3 style="font-size: 24px; margin-bottom: 8px">{{ plan.name }}</h3>
            <div style="font-size: 36px; font-weight: 700; margin-bottom: 4px">{{ plan.price }}</div>
            <p style="color: #64748b; margin-bottom: 24px">{{ plan.desc }}</p>
            <div v-for="f in plan.features" :key="f" style="padding: 8px 0; border-bottom: 1px solid #f1f5f9; color: #475569">
              ✓ {{ f }}
            </div>
            <div style="margin-top: 24px">
              <router-link v-if="plan.link.startsWith('/')" :to="plan.link">
                <NButton :type="plan.featured ? 'primary' : 'default'" block>{{ plan.cta }}</NButton>
              </router-link>
              <a v-else :href="plan.link" target="_blank" style="text-decoration: none">
                <NButton block>{{ plan.cta }}</NButton>
              </a>
            </div>
          </NCard>
        </NGi>
      </NGrid>
    </section>

    <!-- Footer -->
    <footer style="text-align: center; padding: 40px 0; color: #94a3b8; border-top: 1px solid #e2e8f0">
      <NSpace justify="center" :size="24" style="margin-bottom: 12px">
        <a href="https://github.com/tonypk/ai-management-brain" target="_blank" style="color: #6366f1; text-decoration: none">GitHub</a>
        <a href="https://clawhub.ai/tonypk/boss-ai-agent" target="_blank" style="color: #6366f1; text-decoration: none">ClawHub</a>
        <a href="https://t.me/aimanagerbrainbot" target="_blank" style="color: #6366f1; text-decoration: none">Telegram Bot</a>
      </NSpace>
      <p style="font-size: 13px">Built with Claude API. Open source under Apache 2.0.</p>
    </footer>
  </div>
</template>
