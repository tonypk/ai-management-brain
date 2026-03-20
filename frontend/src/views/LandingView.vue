<template>
  <div class="landing">
    <!-- Hero -->
    <section class="hero">
      <h1>AI Management Brain</h1>
      <p class="tagline">Your AI-powered management operating system. Real mentors. Real culture. Real results.</p>
      <div class="hero-cta">
        <a href="#/login" class="btn btn-primary">Get Started Free</a>
        <a href="https://github.com/tonypk/ai-management-brain" class="btn btn-outline" target="_blank">GitHub</a>
      </div>
    </section>

    <!-- Features -->
    <section class="features">
      <h2>What AI Management Brain Does</h2>
      <div class="feature-grid">
        <div class="feature-card">
          <div class="feature-icon">🧠</div>
          <h3>8 Management Mentors</h3>
          <p>Inamori, Dalio, Grove, Ren, Son, Jobs, Bezos, Ma — each with unique check-in questions, chase strategies, and summary styles.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🌏</div>
          <h3>6 Culture Packs</h3>
          <p>Philippines, Singapore, Indonesia, Sri Lanka, Malaysia, China — culturally sensitive communication adapted to your team.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">📊</div>
          <h3>Smart Daily Check-ins</h3>
          <p>AI-powered daily reports with blocker detection, sentiment analysis, and actionable summaries for managers.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">💬</div>
          <h3>Multi-Channel</h3>
          <p>Telegram, Slack, and Lark support. Meet your team where they already are.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🎯</div>
          <h3>Mentor Blending</h3>
          <p>Combine two mentors (e.g., 70% Inamori + 30% Dalio) for a custom management philosophy.</p>
        </div>
        <div class="feature-card">
          <div class="feature-icon">🤖</div>
          <h3>Boss Natural Language</h3>
          <p>"Ask John about project progress" — your AI Chief of Staff translates commands to actions.</p>
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
            <li>2 mentors (Inamori + Dalio)</li>
            <li>2 culture packs (PH + SG)</li>
            <li>Single tenant, up to 10 employees</li>
            <li>Telegram only</li>
            <li>Docker one-click deploy</li>
          </ul>
          <a href="https://github.com/tonypk/ai-management-brain" class="btn btn-outline" target="_blank">Deploy Now</a>
        </div>
        <div class="pricing-card featured">
          <div class="badge">Most Popular</div>
          <h3>Cloud Pro</h3>
          <div class="price">$29<span>/mo</span></div>
          <p class="price-desc">Up to 20 people</p>
          <ul>
            <li>Cloud hosted, zero maintenance</li>
            <li>All 8+ mentors + custom creation</li>
            <li>All 6+ culture packs</li>
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
            <li>Agent orchestration</li>
            <li>All channels + Lark</li>
          </ul>
          <a href="#/login" class="btn btn-outline">Contact Us</a>
        </div>
      </div>
    </section>

    <!-- Footer -->
    <footer class="landing-footer">
      <p>Built with Claude API. Open source on <a href="https://github.com/tonypk/ai-management-brain" target="_blank">GitHub</a>.</p>
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

.hero {
  text-align: center;
  padding: 80px 0 60px;
}
.hero h1 { font-size: 48px; margin-bottom: 16px; }
.tagline { font-size: 20px; color: #666; max-width: 600px; margin: 0 auto 32px; }
.hero-cta { display: flex; gap: 16px; justify-content: center; }

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
.btn-primary {
  background: #2563eb;
  color: white;
  border: 2px solid #2563eb;
}
.btn-primary:hover { background: #1d4ed8; }
.btn-outline {
  background: transparent;
  color: #2563eb;
  border: 2px solid #2563eb;
}
.btn-outline:hover { background: #eff6ff; }

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
.feature-icon { font-size: 40px; margin-bottom: 16px; }
.feature-card h3 { font-size: 20px; margin-bottom: 8px; }
.feature-card p { color: #64748b; line-height: 1.6; }

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
.badge {
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

.landing-footer {
  text-align: center;
  padding: 40px 0;
  color: #94a3b8;
  border-top: 1px solid #e2e8f0;
}
.landing-footer a { color: #2563eb; }
</style>
