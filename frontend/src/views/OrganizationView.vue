<script setup lang="ts">
import { ref, nextTick, onMounted } from 'vue'
import {
  getMentor,
  getOrgPlan,
  startWizard,
  answerWizard,
  adjustOrgPlan,
  activateOrgPlan,
  type MentorInfo,
  type OrgPlan,
} from '../composables/api'

type ViewState = 'loading' | 'start' | 'wizard' | 'plan'

interface ChatMessage {
  role: 'mentor' | 'user'
  content: string
}

const viewState = ref<ViewState>('loading')
const error = ref('')

// --- Start state ---
const mentors = ref<MentorInfo[]>([])
const selectedMentorId = ref('')

// --- Wizard state ---
const messages = ref<ChatMessage[]>([])
const userInput = ref('')
const sending = ref(false)
const chatContainer = ref<HTMLElement | null>(null)

// --- Plan state ---
const orgPlan = ref<OrgPlan | null>(null)
const feedback = ref('')
const adjusting = ref(false)
const activating = ref(false)

async function scrollToBottom() {
  await nextTick()
  if (chatContainer.value) {
    chatContainer.value.scrollTop = chatContainer.value.scrollHeight
  }
}

async function load() {
  error.value = ''
  try {
    const [mentorRes, planRes] = await Promise.allSettled([getMentor(), getOrgPlan()])

    if (mentorRes.status === 'fulfilled') {
      mentors.value = mentorRes.value.data.available_mentors
    }

    if (planRes.status === 'fulfilled') {
      orgPlan.value = planRes.value.data
      viewState.value = 'plan'
    } else {
      viewState.value = 'start'
    }
  } catch (e: any) {
    error.value = e.message
    viewState.value = 'start'
  }
}

async function handleStartWizard() {
  if (!selectedMentorId.value) return
  error.value = ''
  sending.value = true
  try {
    const res = await startWizard(selectedMentorId.value)
    messages.value = [{ role: 'mentor', content: res.data.message }]
    viewState.value = 'wizard'
    await scrollToBottom()
  } catch (e: any) {
    error.value = e.message
  } finally {
    sending.value = false
  }
}

async function handleSendAnswer() {
  const answer = userInput.value.trim()
  if (!answer || sending.value) return

  messages.value = [...messages.value, { role: 'user', content: answer }]
  userInput.value = ''
  sending.value = true
  error.value = ''
  await scrollToBottom()

  try {
    const res = await answerWizard(answer)
    messages.value = [...messages.value, { role: 'mentor', content: res.data.message }]
    await scrollToBottom()

    if (res.data.is_complete && res.data.plan) {
      // Plan generated, reload to show plan view
      const planRes = await getOrgPlan()
      orgPlan.value = planRes.data
      viewState.value = 'plan'
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    sending.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSendAnswer()
  }
}

async function handleAdjust() {
  const fb = feedback.value.trim()
  if (!fb || adjusting.value) return
  adjusting.value = true
  error.value = ''
  try {
    const res = await adjustOrgPlan(fb)
    if (orgPlan.value) {
      orgPlan.value = {
        ...orgPlan.value,
        plan: res.data.plan,
        plan_version: res.data.plan_version,
      }
    }
    feedback.value = ''
  } catch (e: any) {
    error.value = e.message
  } finally {
    adjusting.value = false
  }
}

async function handleActivate() {
  if (activating.value) return
  activating.value = true
  error.value = ''
  try {
    await activateOrgPlan()
    if (orgPlan.value) {
      orgPlan.value = { ...orgPlan.value, status: 'active' }
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    activating.value = false
  }
}

function handleRestart() {
  orgPlan.value = null
  messages.value = []
  viewState.value = 'start'
}

onMounted(load)
</script>

<template>
  <div>
    <h2>Organization Setup</h2>

    <p v-if="error" class="error-msg" style="margin-top: 0.5rem">{{ error }}</p>

    <!-- Loading -->
    <p v-if="viewState === 'loading'" class="loading">Loading...</p>

    <!-- State: Start — Mentor Selector -->
    <template v-if="viewState === 'start'">
      <div class="card" style="margin-top: 1.5rem">
        <h3>Choose a Mentor for Organization Design</h3>
        <p style="color: #888; font-size: 0.9rem; margin-bottom: 1rem">
          Select a management mentor to guide you through designing your organization's structure, culture, and processes.
        </p>
        <div class="mentor-grid">
          <div
            v-for="m in mentors"
            :key="m.id"
            class="mentor-card"
            :class="{ selected: selectedMentorId === m.id }"
            @click="selectedMentorId = m.id"
          >
            <strong style="text-transform: capitalize">{{ m.id }}</strong>
            <div style="font-size: 0.85rem; color: #666; margin-top: 0.25rem">{{ m.name }}</div>
            <div style="font-size: 0.8rem; color: #888; margin-top: 0.25rem">{{ m.description }}</div>
          </div>
        </div>
        <button
          class="btn btn-primary"
          style="margin-top: 1rem"
          :disabled="!selectedMentorId || sending"
          @click="handleStartWizard"
        >
          {{ sending ? 'Starting...' : 'Start Setup Wizard' }}
        </button>
      </div>
    </template>

    <!-- State: Wizard — Chat Interface -->
    <template v-if="viewState === 'wizard'">
      <div class="card chat-card" style="margin-top: 1.5rem">
        <h3>Organization Setup Wizard</h3>
        <div ref="chatContainer" class="chat-container">
          <div
            v-for="(msg, i) in messages"
            :key="i"
            class="chat-msg"
            :class="msg.role === 'mentor' ? 'msg-mentor' : 'msg-user'"
          >
            <div class="msg-label">{{ msg.role === 'mentor' ? 'Mentor' : 'You' }}</div>
            <div class="msg-bubble">{{ msg.content }}</div>
          </div>
          <div v-if="sending" class="chat-msg msg-mentor">
            <div class="msg-label">Mentor</div>
            <div class="msg-bubble thinking">Thinking...</div>
          </div>
        </div>
        <div class="chat-input-row">
          <textarea
            v-model="userInput"
            class="chat-input"
            placeholder="Type your answer..."
            rows="2"
            :disabled="sending"
            @keydown="handleKeydown"
          />
          <button
            class="btn btn-primary"
            :disabled="!userInput.trim() || sending"
            @click="handleSendAnswer"
          >
            Send
          </button>
        </div>
      </div>
    </template>

    <!-- State: Plan — Display -->
    <template v-if="viewState === 'plan' && orgPlan">
      <!-- Status Bar -->
      <div class="card plan-status" style="margin-top: 1.5rem">
        <div style="display: flex; align-items: center; gap: 1rem; flex-wrap: wrap">
          <span
            class="badge"
            :class="orgPlan.status === 'active' ? 'badge-positive' : 'badge-neutral'"
          >
            {{ orgPlan.status.toUpperCase() }}
          </span>
          <span style="color: #888; font-size: 0.85rem">v{{ orgPlan.plan_version }}</span>
          <span style="font-size: 0.85rem">
            {{ orgPlan.industry }} &middot; {{ orgPlan.size }} people &middot; {{ orgPlan.stage }}
          </span>
          <span style="font-size: 0.85rem; color: #888; text-transform: capitalize">
            Mentor: {{ orgPlan.mentor_id }}
          </span>
          <div style="flex: 1" />
          <button
            v-if="orgPlan.status === 'draft'"
            class="btn btn-primary"
            :disabled="activating"
            @click="handleActivate"
          >
            {{ activating ? 'Activating...' : 'Activate Plan' }}
          </button>
          <button class="btn btn-secondary" @click="handleRestart">
            Restart Wizard
          </button>
        </div>
      </div>

      <!-- 1. Management Framework -->
      <div class="card">
        <h3>Management Framework</h3>
        <div class="framework-name">{{ orgPlan.plan.management_framework }}</div>
      </div>

      <!-- 2. Organization Design -->
      <div class="card">
        <h3>Organization Design</h3>
        <p style="margin-bottom: 0.5rem"><strong>Philosophy:</strong> {{ orgPlan.plan.org_design.philosophy }}</p>
        <p style="margin-bottom: 1rem"><strong>Structure:</strong> {{ orgPlan.plan.org_design.structure_type }}</p>

        <h4 style="margin-bottom: 0.5rem; font-size: 0.9rem; color: #555">Units</h4>
        <div class="units-grid">
          <div v-for="unit in orgPlan.plan.org_design.units" :key="unit.name" class="unit-card">
            <strong>{{ unit.name }}</strong>
            <div style="font-size: 0.85rem; color: #666; margin-top: 0.25rem">
              {{ unit.leader_type }}: {{ unit.leader_role }}
            </div>
            <div v-if="unit.size" style="font-size: 0.8rem; color: #888">Size: {{ unit.size }}</div>
            <div v-if="unit.kpis?.length" style="font-size: 0.8rem; color: #888; margin-top: 0.25rem">
              KPIs: {{ unit.kpis.join(', ') }}
            </div>
          </div>
        </div>

        <template v-if="orgPlan.plan.org_design.support_roles?.length">
          <h4 style="margin-top: 1rem; margin-bottom: 0.5rem; font-size: 0.9rem; color: #555">Support Roles</h4>
          <table>
            <thead>
              <tr><th>Title</th><th>Type</th><th>Scope</th></tr>
            </thead>
            <tbody>
              <tr v-for="role in orgPlan.plan.org_design.support_roles" :key="role.title">
                <td>{{ role.title }}</td>
                <td>{{ role.type }}</td>
                <td>{{ role.scope }}</td>
              </tr>
            </tbody>
          </table>
        </template>
      </div>

      <!-- 3. Culture Principles -->
      <div class="card">
        <h3>Culture Principles</h3>
        <ul class="principles-list">
          <li v-for="(p, i) in orgPlan.plan.culture_principles" :key="i">{{ p }}</li>
        </ul>
      </div>

      <!-- 4. KPI System -->
      <div class="card">
        <h3>KPI System</h3>
        <table>
          <thead>
            <tr><th>Name</th><th>Target</th><th>Frequency</th><th>Owner</th></tr>
          </thead>
          <tbody>
            <tr v-for="kpi in orgPlan.plan.kpi_system" :key="kpi.name">
              <td>{{ kpi.name }}</td>
              <td>{{ kpi.target }}</td>
              <td>{{ kpi.frequency }}</td>
              <td>{{ kpi.owner }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 5. Daily Questions -->
      <div class="card">
        <h3>Daily Questions</h3>
        <div v-for="(questions, role) in orgPlan.plan.daily_questions" :key="role" style="margin-bottom: 1rem">
          <h4 style="font-size: 0.9rem; color: #555; text-transform: capitalize; margin-bottom: 0.25rem">{{ role }}</h4>
          <ol class="questions-list">
            <li v-for="(q, i) in questions" :key="i">{{ q }}</li>
          </ol>
        </div>
      </div>

      <!-- 6. Meeting Cadence -->
      <div class="card">
        <h3>Meeting Cadence</h3>
        <table>
          <thead>
            <tr><th>Name</th><th>Frequency</th><th>Duration</th><th>Attendees</th><th>Purpose</th></tr>
          </thead>
          <tbody>
            <tr v-for="m in orgPlan.plan.meeting_cadence" :key="m.name">
              <td>{{ m.name }}</td>
              <td>{{ m.frequency }}</td>
              <td>{{ m.duration }}</td>
              <td>{{ m.attendees }}</td>
              <td>{{ m.purpose }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- 7. Alert Rules -->
      <div class="card">
        <h3>Alert Rules</h3>
        <div v-for="(rule, i) in orgPlan.plan.alert_rules" :key="i" class="alert-card">
          <div><strong>Condition:</strong> {{ rule.condition }}</div>
          <div><strong>Action:</strong> {{ rule.action }}</div>
          <div style="color: #92400e">{{ rule.message }}</div>
        </div>
      </div>

      <!-- 8. AI Reasoning -->
      <div class="card">
        <h3>AI Reasoning</h3>
        <pre class="reasoning-text">{{ orgPlan.plan.reasoning }}</pre>
      </div>

      <!-- Adjustment Area (draft only) -->
      <div v-if="orgPlan.status === 'draft'" class="card">
        <h3>Adjust Plan</h3>
        <p style="color: #888; font-size: 0.9rem; margin-bottom: 0.75rem">
          Provide feedback to adjust the plan. The AI mentor will revise it accordingly.
        </p>
        <textarea
          v-model="feedback"
          class="adjust-input"
          placeholder="e.g., I want a flatter structure with fewer management layers..."
          rows="3"
        />
        <button
          class="btn btn-primary"
          style="margin-top: 0.5rem"
          :disabled="!feedback.trim() || adjusting"
          @click="handleAdjust"
        >
          {{ adjusting ? 'Adjusting...' : 'Submit Feedback' }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.mentor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
}
.mentor-card {
  padding: 1rem;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}
.mentor-card:hover { border-color: #6366f1; }
.mentor-card.selected { border-color: #6366f1; background: rgba(99,102,241,0.05); }

/* Chat */
.chat-card { display: flex; flex-direction: column; }
.chat-container {
  display: flex;
  flex-direction: column;
  height: 60vh;
  overflow-y: auto;
  padding: 1rem 0;
  gap: 0.75rem;
}
.chat-msg { display: flex; flex-direction: column; max-width: 80%; }
.msg-mentor { align-self: flex-start; }
.msg-user { align-self: flex-end; }
.msg-label { font-size: 0.75rem; color: #888; margin-bottom: 0.2rem; }
.msg-user .msg-label { text-align: right; }
.msg-bubble {
  padding: 0.75rem 1rem;
  border-radius: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}
.msg-mentor .msg-bubble { background: #e0e7ff; color: #1e1b4b; }
.msg-user .msg-bubble { background: #6366f1; color: #fff; }
.thinking { opacity: 0.7; font-style: italic; }
.chat-input-row {
  display: flex;
  gap: 0.5rem;
  align-items: flex-end;
  padding-top: 0.75rem;
  border-top: 1px solid #eee;
}
.chat-input {
  flex: 1;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
  font-family: inherit;
  resize: none;
  outline: none;
  transition: border-color 0.2s;
}
.chat-input:focus { border-color: #6366f1; }

/* Plan */
.plan-status { background: #fff; }
.framework-name {
  font-size: 1.5rem;
  font-weight: 700;
  color: #6366f1;
}
.units-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 0.75rem;
}
.unit-card {
  padding: 0.75rem;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fafafa;
}
.principles-list {
  list-style: disc;
  padding-left: 1.5rem;
}
.principles-list li { margin-bottom: 0.35rem; line-height: 1.5; }
.questions-list {
  padding-left: 1.5rem;
}
.questions-list li { margin-bottom: 0.25rem; line-height: 1.5; }
.alert-card {
  padding: 0.75rem 1rem;
  border-left: 4px solid #f59e0b;
  background: #fffbeb;
  border-radius: 0 8px 8px 0;
  margin-bottom: 0.5rem;
  line-height: 1.5;
}
.reasoning-text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.9rem;
  line-height: 1.6;
  color: #444;
  background: #f9fafb;
  padding: 1rem;
  border-radius: 6px;
  font-family: inherit;
}
.adjust-input {
  width: 100%;
  padding: 0.5rem 0.75rem;
  border: 1px solid #ddd;
  border-radius: 6px;
  font-size: 0.9rem;
  font-family: inherit;
  resize: vertical;
  outline: none;
  transition: border-color 0.2s;
}
.adjust-input:focus { border-color: #6366f1; }

/* Mobile */
@media (max-width: 768px) {
  .chat-container { height: 50vh; }
  .chat-msg { max-width: 90%; }
  .mentor-grid { grid-template-columns: 1fr; }
  .units-grid { grid-template-columns: 1fr; }
  table { font-size: 0.8rem; }
  th, td { padding: 0.5rem; }
}
</style>
