<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getMentor, updateMentor, updateBlend, type MentorConfig } from '../composables/api'

const config = ref<MentorConfig | null>(null)
const loading = ref(true)
const error = ref('')
const success = ref('')

const blendMode = ref(false)
const selectedMentor = ref('')
const blendPrimary = ref('')
const blendSecondary = ref('')
const blendWeight = ref(70)
const saving = ref(false)

async function loadMentor() {
  try {
    const res = await getMentor()
    config.value = res.data
    selectedMentor.value = res.data.current_mentor_id
    if (res.data.current_blend) {
      blendMode.value = true
      blendPrimary.value = res.data.current_blend.primary_id
      blendSecondary.value = res.data.current_blend.secondary_id
      blendWeight.value = Math.round(res.data.current_blend.weight * 100)
    }
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    if (blendMode.value) {
      await updateBlend(blendPrimary.value, blendSecondary.value, blendWeight.value)
      success.value = `Blend saved: ${blendPrimary.value} (${blendWeight.value}%) + ${blendSecondary.value} (${100 - blendWeight.value}%)`
    } else {
      await updateMentor(selectedMentor.value)
      success.value = `Mentor switched to ${selectedMentor.value}`
    }
    await loadMentor()
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}

onMounted(loadMentor)
</script>

<template>
  <div>
    <h2>Mentor Configuration</h2>

    <p v-if="loading" class="loading">Loading...</p>
    <template v-else-if="config">
      <div class="card" style="margin-top: 1.5rem">
        <h3>Current Mentor: <span style="color: #6366f1; text-transform: capitalize">{{ config.current_mentor_id }}</span></h3>
        <p v-if="config.current_blend" style="color: #888; font-size: 0.9rem; margin-top: 0.25rem">
          Blend: {{ config.current_blend.primary_id }} ({{ Math.round(config.current_blend.weight * 100) }}%)
          + {{ config.current_blend.secondary_id }} ({{ Math.round((1 - config.current_blend.weight) * 100) }}%)
        </p>
      </div>

      <div class="card">
        <h3>Available Mentors</h3>
        <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 1rem; margin-bottom: 1.5rem">
          <div
            v-for="m in config.available_mentors"
            :key="m.id"
            class="mentor-card"
            :class="{ selected: selectedMentor === m.id && !blendMode }"
            @click="blendMode || (selectedMentor = m.id)"
          >
            <strong style="text-transform: capitalize">{{ m.id }}</strong>
            <div style="font-size: 0.85rem; color: #666; margin-top: 0.25rem">{{ m.name }}</div>
            <div style="font-size: 0.8rem; color: #888; margin-top: 0.25rem">{{ m.description }}</div>
          </div>
        </div>

        <div style="margin-bottom: 1rem">
          <label style="display: flex; align-items: center; gap: 0.5rem; cursor: pointer">
            <input type="checkbox" v-model="blendMode" />
            <span>Enable Mentor Blending</span>
          </label>
        </div>

        <div v-if="blendMode" style="display: flex; gap: 1rem; flex-wrap: wrap; margin-bottom: 1rem">
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Primary Mentor</label>
            <select v-model="blendPrimary">
              <option v-for="m in config.available_mentors" :key="m.id" :value="m.id">{{ m.name }}</option>
            </select>
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">Secondary Mentor</label>
            <select v-model="blendSecondary">
              <option v-for="m in config.available_mentors" :key="m.id" :value="m.id">{{ m.name }}</option>
            </select>
          </div>
          <div>
            <label style="display: block; font-size: 0.85rem; color: #666; margin-bottom: 0.25rem">
              Primary Weight: {{ blendWeight }}%
            </label>
            <input type="range" v-model.number="blendWeight" min="50" max="90" style="width: 200px" />
          </div>
        </div>

        <p v-if="error" class="error-msg">{{ error }}</p>
        <p v-if="success" style="color: #065f46; font-size: 0.85rem; margin-bottom: 0.5rem">{{ success }}</p>

        <button class="btn btn-primary" @click="handleSave" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Configuration' }}
        </button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.mentor-card {
  padding: 1rem;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}
.mentor-card:hover { border-color: #6366f1; }
.mentor-card.selected { border-color: #6366f1; background: rgba(99,102,241,0.05); }
</style>
