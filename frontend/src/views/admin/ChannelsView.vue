<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  getChannelConfig,
  updateChannelConfig,
  testChannel,
  type ChannelConfig,
} from '../../composables/api'

const config = ref<ChannelConfig | null>(null)
const loading = ref(true)
const error = ref('')
const success = ref('')
const saving = ref(false)

// Form state
const enabledChannels = ref<string[]>([])
const slackBotToken = ref('')
const slackSigningSecret = ref('')
const larkAppId = ref('')
const larkAppSecret = ref('')
const signalPhone = ref('')

// Test modal
const showTestModal = ref(false)
const testChannelName = ref('')
const testUserId = ref('')
const testText = ref('')
const testing = ref(false)
const testResult = ref('')

const channelLabels: Record<string, string> = {
  telegram: 'Telegram',
  signal: 'Signal',
  slack: 'Slack',
  lark: 'Lark',
}

async function loadConfig() {
  try {
    const res = await getChannelConfig()
    config.value = res.data
    enabledChannels.value = [...res.data.enabled_channels]
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function isConfigured(channelType: string): boolean {
  if (!config.value) return false
  const ch = config.value.channels.find((c) => c.type === channelType)
  return ch?.configured ?? false
}

function isEnabled(channelType: string): boolean {
  return enabledChannels.value.includes(channelType)
}

function toggleChannel(channelType: string) {
  if (channelType === 'telegram') return // always active
  const idx = enabledChannels.value.indexOf(channelType)
  if (idx >= 0) {
    enabledChannels.value = enabledChannels.value.filter((c) => c !== channelType)
  } else {
    enabledChannels.value = [...enabledChannels.value, channelType]
  }
}

async function handleSave() {
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    await updateChannelConfig({
      enabled_channels: enabledChannels.value,
      slack_bot_token: slackBotToken.value || undefined,
      slack_signing_secret: slackSigningSecret.value || undefined,
      lark_app_id: larkAppId.value || undefined,
      lark_app_secret: larkAppSecret.value || undefined,
      signal_phone: signalPhone.value || undefined,
    })
    success.value = 'Channel configuration saved successfully.'
    await loadConfig()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

function openTestModal(channel: string) {
  testChannelName.value = channel
  testUserId.value = ''
  testText.value = ''
  testResult.value = ''
  showTestModal.value = true
}

async function handleTest() {
  testing.value = true
  testResult.value = ''
  try {
    const res = await testChannel(testChannelName.value, testUserId.value, testText.value || undefined)
    testResult.value = res.data.sent ? 'Test message sent successfully!' : 'Failed to send test message.'
  } catch (e: any) {
    testResult.value = `Error: ${e.message}`
  } finally {
    testing.value = false
  }
}

onMounted(loadConfig)
</script>

<template>
  <div>
    <h2>Channel Configuration</h2>

    <p v-if="loading" class="loading">Loading...</p>
    <p v-else-if="error && !config" class="error-msg">{{ error }}</p>
    <template v-else-if="config">
      <div class="stats-grid" style="margin-top: 1.5rem">
        <!-- Telegram Card -->
        <div class="card channel-card">
          <div class="channel-header">
            <span class="channel-name">Telegram</span>
            <span class="status-dot configured"></span>
          </div>
          <p class="channel-info">Always active (primary channel)</p>
          <div class="channel-actions">
            <button class="btn btn-secondary btn-sm" @click="openTestModal('telegram')">Test</button>
          </div>
        </div>

        <!-- Signal Card -->
        <div class="card channel-card">
          <div class="channel-header">
            <span class="channel-name">Signal</span>
            <span class="status-dot" :class="isConfigured('signal') ? 'configured' : 'not-configured'"></span>
          </div>
          <label class="toggle-label">
            <input type="checkbox" :checked="isEnabled('signal')" @change="toggleChannel('signal')" />
            <span>{{ isEnabled('signal') ? 'Enabled' : 'Disabled' }}</span>
          </label>
          <div style="margin-top: 0.75rem">
            <label class="field-label">Phone Number</label>
            <input v-model="signalPhone" placeholder="+1234567890" style="width: 100%" />
          </div>
          <div class="channel-actions">
            <button class="btn btn-secondary btn-sm" :disabled="!isConfigured('signal')" @click="openTestModal('signal')">Test</button>
          </div>
        </div>

        <!-- Slack Card -->
        <div class="card channel-card">
          <div class="channel-header">
            <span class="channel-name">Slack</span>
            <span class="status-dot" :class="isConfigured('slack') ? 'configured' : 'not-configured'"></span>
          </div>
          <label class="toggle-label">
            <input type="checkbox" :checked="isEnabled('slack')" @change="toggleChannel('slack')" />
            <span>{{ isEnabled('slack') ? 'Enabled' : 'Disabled' }}</span>
          </label>
          <div style="margin-top: 0.75rem">
            <label class="field-label">Bot Token</label>
            <input v-model="slackBotToken" type="password" placeholder="xoxb-..." style="width: 100%" />
          </div>
          <div style="margin-top: 0.5rem">
            <label class="field-label">Signing Secret</label>
            <input v-model="slackSigningSecret" type="password" placeholder="Signing secret" style="width: 100%" />
          </div>
          <div class="channel-actions">
            <button class="btn btn-secondary btn-sm" :disabled="!isConfigured('slack')" @click="openTestModal('slack')">Test</button>
          </div>
        </div>

        <!-- Lark Card -->
        <div class="card channel-card">
          <div class="channel-header">
            <span class="channel-name">Lark</span>
            <span class="status-dot" :class="isConfigured('lark') ? 'configured' : 'not-configured'"></span>
          </div>
          <label class="toggle-label">
            <input type="checkbox" :checked="isEnabled('lark')" @change="toggleChannel('lark')" />
            <span>{{ isEnabled('lark') ? 'Enabled' : 'Disabled' }}</span>
          </label>
          <div style="margin-top: 0.75rem">
            <label class="field-label">App ID</label>
            <input v-model="larkAppId" placeholder="App ID" style="width: 100%" />
          </div>
          <div style="margin-top: 0.5rem">
            <label class="field-label">App Secret</label>
            <input v-model="larkAppSecret" type="password" placeholder="App Secret" style="width: 100%" />
          </div>
          <div class="channel-actions">
            <button class="btn btn-secondary btn-sm" :disabled="!isConfigured('lark')" @click="openTestModal('lark')">Test</button>
          </div>
        </div>
      </div>

      <p v-if="error" class="error-msg">{{ error }}</p>
      <p v-if="success" style="color: #065f46; font-size: 0.85rem; margin-bottom: 0.5rem">{{ success }}</p>

      <button class="btn btn-primary" @click="handleSave" :disabled="saving">
        {{ saving ? 'Saving...' : 'Save Configuration' }}
      </button>
    </template>

    <!-- Test Modal -->
    <div v-if="showTestModal" class="modal-overlay" @click.self="showTestModal = false">
      <div class="modal-content card">
        <h3>Test {{ channelLabels[testChannelName] || testChannelName }}</h3>
        <div style="margin-top: 1rem">
          <label class="field-label">User ID</label>
          <input v-model="testUserId" placeholder="User ID or phone number" style="width: 100%" />
        </div>
        <div style="margin-top: 0.75rem">
          <label class="field-label">Message (optional)</label>
          <input v-model="testText" placeholder="Test message" style="width: 100%" />
        </div>
        <p v-if="testResult" :class="testResult.startsWith('Error') ? 'error-msg' : 'success-msg'" style="margin-top: 0.75rem">
          {{ testResult }}
        </p>
        <div style="margin-top: 1rem; display: flex; gap: 0.5rem">
          <button class="btn btn-primary" @click="handleTest" :disabled="testing || !testUserId">
            {{ testing ? 'Sending...' : 'Send Test' }}
          </button>
          <button class="btn btn-secondary" @click="showTestModal = false">Close</button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.channel-card {
  display: flex;
  flex-direction: column;
  min-height: 200px;
}
.channel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}
.channel-name {
  font-weight: 700;
  font-size: 1.1rem;
}
.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
}
.status-dot.configured {
  background: #10b981;
}
.status-dot.not-configured {
  background: #ef4444;
}
.channel-info {
  color: #888;
  font-size: 0.85rem;
}
.channel-actions {
  margin-top: auto;
  padding-top: 0.75rem;
}
.toggle-label {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  cursor: pointer;
  font-size: 0.9rem;
}
.field-label {
  display: block;
  font-size: 0.85rem;
  color: #666;
  margin-bottom: 0.25rem;
}
.btn-sm {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}
.modal-content {
  width: 400px;
  max-width: 90vw;
}
.success-msg {
  color: #065f46;
  font-size: 0.85rem;
}
</style>
