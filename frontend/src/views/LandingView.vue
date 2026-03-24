<template>
  <div class="landing">
    <!-- Hero -->
    <section class="hero">
      <h1>AI Management Brain</h1>
      <p class="tagline">Your AI-powered management operating system. Assemble a virtual C-Suite, choose from 16 mentor philosophies, and manage across cultures — in Telegram, Slack, or your AI coding tools.</p>
      <div class="hero-cta">
        <a href="#/login" class="btn btn-primary">Get Started Free</a>
        <a href="https://github.com/tonypk/ai-management-brain" class="btn btn-outline" target="_blank">GitHub</a>
      </div>
      <div class="hero-badges">
        <span class="hero-badge">Open Source</span>
        <span class="hero-badge">ClawHub Skill</span>
        <span class="hero-badge">Claude Cowork Plugin</span>
      </div>
    </section>

    <!-- Integrations banner -->
    <section class="integrations">
      <h2>Works With Your Tools</h2>
      <div class="integration-grid">
        <div class="integration-item">
          <div class="integration-icon">🔌</div>
          <h4>ClawHub</h4>
          <p>Install as an OpenClaw skill. One command, zero config.</p>
          <code>clawhub install boss-ai-agent</code>
        </div>
        <div class="integration-item">
          <div class="integration-icon">🤝</div>
          <h4>Claude Cowork</h4>
          <p>Add as a Claude Code plugin. Your AI mentor lives inside your IDE.</p>
          <code>/boss-ai-agent</code>
        </div>
        <div class="integration-item">
          <div class="integration-icon">🤖</div>
          <h4>Telegram Bot</h4>
          <p>Full-featured Telegram bot for team check-ins, reports, and board discussions.</p>
          <code>@aimanagerbrainbot</code>
        </div>
        <div class="integration-item">
          <div class="integration-icon">🌐</div>
          <h4>Web Dashboard</h4>
          <p>Full management dashboard with analytics, team management, and AI C-Suite controls.</p>
          <code>manageaibrain.com</code>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section class="features">
      <h2>Core Features</h2>
      <div class="feature-grid">
        <div class="feature-card highlight">
          <div class="feature-icon">👔</div>
          <h3>AI C-Suite</h3>
          <p>Assemble your virtual management team — CEO, CFO, CMO, CTO, CHRO, COO. Assign expert personas to each seat, then run board discussions where every seat weighs in on your strategic questions.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🧠</div>
          <h3>16 Management Mentors</h3>
          <p>Inamori, Dalio, Grove, Ren, Son, Jobs, Bezos, Ma, Musk, Buffett, plus domain experts — Jack Trout (positioning strategy) and Erin Meyer (cross-cultural leadership). Each with unique philosophies.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🌏</div>
          <h3>9 Culture Packs</h3>
          <p>Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China, USA, India, and default — culturally adapted communication for global teams.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">📊</div>
          <h3>Smart Daily Check-ins</h3>
          <p>AI-powered daily reports with blocker detection, sentiment analysis, and actionable summaries. 7 automated scenarios from check-in to emergency response.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">💬</div>
          <h3>Multi-Channel</h3>
          <p>Telegram, Slack, and Lark support. Meet your team where they already are. Or use the ClawHub skill to manage from any OpenClaw-connected platform.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🎯</div>
          <h3>Mentor Blending</h3>
          <p>Combine two mentors (e.g., 70% Inamori + 30% Dalio) for a custom management philosophy that fits your leadership style.</p>
        </div>
      </div>
    </section>

    <!-- C-Suite deep dive -->
    <section class="csuite">
      <h2>Your Virtual Board of Directors</h2>
      <p class="section-desc">Assign the right expert to each seat. Run board discussions on any topic. Get synthesized strategic advice from multiple perspectives.</p>
      <div class="csuite-grid">
        <div class="seat-card">
          <div class="seat-type">CEO</div>
          <div class="seat-desc">Strategic vision and execution</div>
        </div>
        <div class="seat-card">
          <div class="seat-type">CFO</div>
          <div class="seat-desc">Financial strategy and risk</div>
        </div>
        <div class="seat-card">
          <div class="seat-type">CMO</div>
          <div class="seat-desc">Marketing and positioning</div>
        </div>
        <div class="seat-card">
          <div class="seat-type">CTO</div>
          <div class="seat-desc">Technology and architecture</div>
        </div>
        <div class="seat-card">
          <div class="seat-type">CHRO</div>
          <div class="seat-desc">People and culture</div>
        </div>
        <div class="seat-card">
          <div class="seat-type">COO</div>
          <div class="seat-desc">Operations and processes</div>
        </div>
      </div>
      <div class="csuite-example">
        <div class="example-label">Example: Board Discussion</div>
        <div class="example-content">
          <div class="example-cmd">/board Should we expand to the Japan market?</div>
          <div class="example-responses">
            <div class="example-resp"><strong>CEO (Musk):</strong> First principles — what's the TAM and can we 10x there?</div>
            <div class="example-resp"><strong>CFO (Dalio):</strong> Risk-adjusted ROI needs modeling. What's the capital requirement?</div>
            <div class="example-resp"><strong>CMO (Trout):</strong> Position as the first AI management tool for Japanese enterprise...</div>
            <div class="example-resp"><strong>CHRO (Meyer):</strong> Japanese high-context culture requires relationship-first approach...</div>
            <div class="example-resp synthesis"><strong>Synthesis:</strong> Expand — but hire a local GM first. Budget $200K for 6-month pilot.</div>
          </div>
        </div>
      </div>
    </section>

    <!-- Skill / Plugin -->
    <section class="skill-section">
      <h2>Install in Seconds</h2>
      <div class="install-grid">
        <div class="install-card">
          <h3>ClawHub (OpenClaw Skill)</h3>
          <p>Full management middleware as an OpenClaw skill. 7 automated scenarios, 14 mentors, 9 cultures. Works with any platform connected to your OpenClaw gateway.</p>
          <div class="install-cmd">
            <code>clawhub install boss-ai-agent</code>
          </div>
          <ul class="install-features">
            <li>7 automated scenarios (check-in, patrol, briefing, 1:1, signals, KB, emergency)</li>
            <li>Works offline — zero cloud dependency</li>
            <li>Optional cloud API for full mentor configs and analytics</li>
          </ul>
        </div>
        <div class="install-card">
          <h3>Claude Code / Cowork Plugin</h3>
          <p>Your AI management mentor inside Claude Code. Get strategic advice, run board discussions, and manage your team while you code.</p>
          <div class="install-cmd">
            <code>/install-skill management-brain</code>
          </div>
          <ul class="install-features">
            <li>In-editor management mentor</li>
            <li>Board discussions from your terminal</li>
            <li>Connects to manageaibrain.com cloud</li>
          </ul>
        </div>
      </div>
    </section>

    <!-- Pricing -->
    <section class="pricing" id="pricing">
      <h2>Simple Pricing</h2>
      <div class="pricing-grid">
        <div class="pricing-card">
          <h3>Open Source</h3>
          <div class="price">Free</div>
          <p class="price-desc">Self-hosted, Apache 2.0</p>
          <ul>
            <li>Complete Bot + API + Dashboard</li>
            <li>16 mentors + 9 culture packs</li>
            <li>AI C-Suite with board discussions</li>
            <li>Single tenant, up to 10 employees</li>
            <li>ClawHub skill included</li>
            <li>Docker one-click deploy</li>
          </ul>
          <a href="https://github.com/tonypk/ai-management-brain" class="btn btn-outline" target="_blank">Deploy Now</a>
        </div>
        <div class="pricing-card featured">
          <div class="featured-badge">Most Popular</div>
          <h3>Cloud Pro</h3>
          <div class="price">$29<span>/mo</span></div>
          <p class="price-desc">Up to 20 people</p>
          <ul>
            <li>Cloud hosted, zero maintenance</li>
            <li>All mentors + custom creation</li>
            <li>Full C-Suite + board discussions</li>
            <li>Multi-channel (Telegram + Slack)</li>
            <li>Mentor blending</li>
            <li>Analytics dashboard</li>
          </ul>
          <a href="#/login" class="btn btn-primary">Start Free Trial</a>
        </div>
        <div class="pricing-card">
          <h3>Enterprise</h3>
          <div class="price">$299<span>/mo</span></div>
          <p class="price-desc">Up to 100 people</p>
          <ul>
            <li>Everything in Pro</li>
            <li>Multi-tenant (departments)</li>
            <li>BYOK (your own API key)</li>
            <li>SSO + RBAC</li>
            <li>Custom mentors and culture packs</li>
            <li>All channels + Lark + Cowork</li>
          </ul>
          <a href="#/login" class="btn btn-outline">Contact Us</a>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="landing-footer">
      <div class="footer-links">
        <a href="https://github.com/tonypk/ai-management-brain" target="_blank">GitHub</a>
        <span class="footer-sep">|</span>
        <a href="https://clawhub.ai/tonypk/boss-ai-agent" target="_blank">ClawHub</a>
        <span class="footer-sep">|</span>
        <a href="https://t.me/aimanagerbrainbot" target="_blank">Telegram Bot</a>
      </div>
      <p>Built with Claude API. Open source under Apache 2.0.</p>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  max-width: 1200px;
  margin: 0 auto;
  padding: 0 24px;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
}

/* Hero */
.hero {
  text-align: center;
  padding: 80px 0 40px;
}
.hero h1 { font-size: 48px; margin-bottom: 16px; }
.tagline { font-size: 20px; color: #666; max-width: 700px; margin: 0 auto 32px; line-height: 1.6; }
.hero-cta { display: flex; gap: 16px; justify-content: center; }
.hero-badges { display: flex; gap: 12px; justify-content: center; margin-top: 24px; }
.hero-badge {
  background: #f0f9ff;
  color: #0369a1;
  padding: 6px 16px;
  border-radius: 20px;
  font-size: 13px;
  font-weight: 600;
  border: 1px solid #bae6fd;
}

.btn {
  display: inline-block;
  padding: 12px 32px;
  border-radius: 8px;
  font-size: 16px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.2s;
}
.btn-primary { background: #2563eb; color: white; border: 2px solid #2563eb; }
.btn-primary:hover { background: #1d4ed8; }
.btn-outline { background: transparent; color: #2563eb; border: 2px solid #2563eb; }
.btn-outline:hover { background: #eff6ff; }

/* Integrations */
.integrations { padding: 40px 0 60px; }
.integrations h2 { text-align: center; font-size: 28px; margin-bottom: 32px; }
.integration-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
  gap: 20px;
}
.integration-item {
  text-align: center;
  padding: 24px 16px;
  border-radius: 12px;
  border: 1px solid #e2e8f0;
  background: #fafbfc;
}
.integration-icon { font-size: 36px; margin-bottom: 12px; }
.integration-item h4 { font-size: 16px; margin-bottom: 6px; }
.integration-item p { font-size: 13px; color: #64748b; line-height: 1.5; margin-bottom: 12px; }
.integration-item code {
  display: inline-block;
  background: #1e293b;
  color: #a5f3fc;
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 12px;
}

/* Features */
.features { padding: 60px 0; }
.features h2 { text-align: center; font-size: 32px; margin-bottom: 40px; }
.feature-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 24px;
}
.feature-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 32px;
  border: 1px solid #e2e8f0;
}
.feature-card.highlight {
  background: linear-gradient(135deg, #eff6ff 0%, #f5f3ff 100%);
  border-color: #a5b4fc;
}
.feature-icon { font-size: 40px; margin-bottom: 16px; }
.feature-card h3 { font-size: 20px; margin-bottom: 8px; }
.feature-card p { color: #64748b; line-height: 1.6; }

/* C-Suite */
.csuite { padding: 60px 0; }
.csuite h2 { text-align: center; font-size: 32px; margin-bottom: 8px; }
.section-desc { text-align: center; color: #64748b; max-width: 600px; margin: 0 auto 36px; line-height: 1.6; }
.csuite-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 16px;
  margin-bottom: 40px;
}
.seat-card {
  background: #1e293b;
  color: white;
  border-radius: 12px;
  padding: 24px 16px;
  text-align: center;
}
.seat-type { font-size: 24px; font-weight: 700; margin-bottom: 4px; }
.seat-desc { font-size: 12px; color: #94a3b8; }

.csuite-example {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 28px;
  max-width: 720px;
  margin: 0 auto;
}
.example-label { font-size: 12px; font-weight: 600; color: #6366f1; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 16px; }
.example-cmd {
  background: #1e293b;
  color: #a5f3fc;
  padding: 10px 16px;
  border-radius: 8px;
  font-family: monospace;
  font-size: 14px;
  margin-bottom: 16px;
}
.example-responses { display: flex; flex-direction: column; gap: 10px; }
.example-resp { font-size: 14px; color: #475569; line-height: 1.5; padding: 8px 12px; background: white; border-radius: 8px; border: 1px solid #e2e8f0; }
.example-resp.synthesis { background: #f0fdf4; border-color: #86efac; }

/* Skill / Plugin */
.skill-section { padding: 60px 0; }
.skill-section h2 { text-align: center; font-size: 32px; margin-bottom: 36px; }
.install-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(340px, 1fr));
  gap: 24px;
  max-width: 800px;
  margin: 0 auto;
}
.install-card {
  background: white;
  border-radius: 12px;
  padding: 28px;
  border: 2px solid #e2e8f0;
}
.install-card h3 { font-size: 20px; margin-bottom: 8px; }
.install-card > p { color: #64748b; font-size: 14px; line-height: 1.5; margin-bottom: 16px; }
.install-cmd {
  background: #1e293b;
  padding: 12px 16px;
  border-radius: 8px;
  margin-bottom: 16px;
}
.install-cmd code { color: #a5f3fc; font-size: 14px; }
.install-features { list-style: none; padding: 0; margin: 0; }
.install-features li {
  padding: 6px 0;
  color: #475569;
  font-size: 14px;
  border-bottom: 1px solid #f1f5f9;
}
.install-features li::before { content: "✓ "; color: #22c55e; font-weight: bold; }

/* Pricing */
.pricing { padding: 60px 0; }
.pricing h2 { text-align: center; font-size: 32px; margin-bottom: 40px; }
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 24px;
  max-width: 960px;
  margin: 0 auto;
}
.pricing-card {
  background: white;
  border-radius: 12px;
  padding: 32px;
  border: 2px solid #e2e8f0;
  position: relative;
}
.pricing-card.featured {
  border-color: #2563eb;
  box-shadow: 0 4px 24px rgba(37, 99, 235, 0.15);
}
.featured-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: #2563eb;
  color: white;
  padding: 4px 16px;
  border-radius: 20px;
  font-size: 12px;
  font-weight: 600;
}
.pricing-card h3 { font-size: 24px; margin-bottom: 8px; }
.price { font-size: 48px; font-weight: 700; margin-bottom: 4px; }
.price span { font-size: 16px; color: #666; }
.price-desc { color: #64748b; margin-bottom: 24px; }
.pricing-card ul { list-style: none; padding: 0; margin: 0 0 24px; }
.pricing-card li {
  padding: 8px 0;
  border-bottom: 1px solid #f1f5f9;
  color: #475569;
}
.pricing-card li::before { content: "✓ "; color: #22c55e; font-weight: bold; }
.pricing-card .btn { width: 100%; text-align: center; box-sizing: border-box; }

/* Footer */
.landing-footer {
  text-align: center;
  padding: 40px 0;
  color: #94a3b8;
  border-top: 1px solid #e2e8f0;
}
.footer-links { margin-bottom: 12px; }
.footer-links a { color: #2563eb; text-decoration: none; font-size: 14px; }
.footer-links a:hover { text-decoration: underline; }
.footer-sep { color: #cbd5e1; margin: 0 12px; }
.landing-footer p { font-size: 13px; }

/* Responsive */
@media (max-width: 768px) {
  .hero h1 { font-size: 32px; }
  .tagline { font-size: 16px; }
  .hero-cta { flex-direction: column; align-items: center; }
  .hero-badges { flex-wrap: wrap; }
  .csuite-grid { grid-template-columns: repeat(3, 1fr); }
  .install-grid { grid-template-columns: 1fr; }
}
</style>
