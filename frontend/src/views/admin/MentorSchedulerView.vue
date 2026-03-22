<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  listAllMentors,
  updateMentor,
  getMentor,
  listSchedulerJobs,
  updateJobSchedule,
  triggerJob,
  type MentorInfo,
  type SchedulerJob,
} from '../../composables/api'

// Mentor state
const mentors = ref<MentorInfo[]>([])
const currentMentorId = ref('')
const mentorLoading = ref(true)
const mentorError = ref('')
const mentorSuccess = ref('')
const switchingMentor = ref(false)

// Scheduler state
const jobs = ref<SchedulerJob[]>([])
const schedulerLoading = ref(true)
const schedulerError = ref('')
const schedulerSuccess = ref('')

// Track edited cron values
const editedCrons = ref<Record<string, string>>({})
const savingJob = ref('')
const triggeringJob = ref('')

async function loadMentors() {
  mentorLoading.value = true
  mentorError.value = ''
  try {
    const [mentorsRes, configRes] = await Promise.allSettled([
      listAllMentors(),
      getMentor(),
    ])
    if (mentorsRes.status === 'fulfilled') {
      mentors.value = mentorsRes.value.data
    }
    if (configRes.status === 'fulfilled') {
      currentMentorId.value = configRes.value.data.current_mentor_id
    }
  } catch (e: any) {
    mentorError.value = e.message
  } finally {
    mentorLoading.value = false
  }
}

async function selectMentor(mentorId: string) {
  if (mentorId === currentMentorId.value) return
  switchingMentor.value = true
  mentorError.value = ''
  mentorSuccess.value = ''
  try {
    await updateMentor(mentorId)
    currentMentorId.value = mentorId
    mentorSuccess.value = `Switched to ${mentorId}`
  } catch (e: any) {
    mentorError.value = e.message
  } finally {
    switchingMentor.value = false
  }
}

async function loadJobs() {
  schedulerLoading.value = true
  schedulerError.value = ''
  try {
    const res = await listSchedulerJobs()
    jobs.value = res.data
    // Initialize edited crons
    const crons: Record<string, string> = {}
    for (const j of res.data) {
      crons[j.name] = j.cron
    }
    editedCrons.value = crons
  } catch (e: any) {
    schedulerError.value = e.message
  } finally {
    schedulerLoading.value = false
  }
}

function cronChanged(jobName: string): boolean {
  const job = jobs.value.find((j) => j.name === jobName)
  return job ? editedCrons.value[jobName] !== job.cron : false
}

async function handleSaveJob(jobName: string) {
  savingJob.value = jobName
  schedulerError.value = ''
  schedulerSuccess.value = ''
  try {
    await updateJobSchedule(jobName, editedCrons.value[jobName])
    schedulerSuccess.value = `Schedule updated for ${jobName}`
    await loadJobs()
  } catch (e: any) {
    schedulerError.value = e.message
  } finally {
    savingJob.value = ''
  }
}

async function handleTrigger(jobName: string) {
  triggeringJob.value = jobName
  schedulerError.value = ''
  schedulerSuccess.value = ''
  try {
    await triggerJob(jobName)
    schedulerSuccess.value = `${jobName} triggered successfully`
  } catch (e: any) {
    schedulerError.value = e.message
  } finally {
    triggeringJob.value = ''
  }
}

function formatTime(ts: string): string {
  if (!ts) return '-'
  return new Date(ts).toLocaleString()
}

onMounted(() => {
  loadMentors()
  loadJobs()
})
</script>

<template>
  <div>
    <h2>Mentor & Scheduler</h2>

    <!-- Mentor Section -->
    <div class="card" style="margin-top: 1.5rem">
      <h3>Active Mentor: <span style="color: #6366f1; text-transform: capitalize">{{ currentMentorId || '...' }}</span></h3>

      <p v-if="mentorLoading" class="loading">Loading mentors...</p>
      <p v-if="mentorError" class="error-msg">{{ mentorError }}</p>
      <p v-if="mentorSuccess" style="color: #065f46; font-size: 0.85rem; margin-bottom: 0.5rem">{{ mentorSuccess }}</p>

      <div v-if="!mentorLoading" class="mentor-grid">
        <div
          v-for="m in mentors"
          :key="m.id"
          class="mentor-card"
          :class="{ selected: currentMentorId === m.id, disabled: switchingMentor }"
          @click="selectMentor(m.id)"
        >
          <strong style="text-transform: capitalize">{{ m.id }}</strong>
          <div style="font-size: 0.85rem; color: #666; margin-top: 0.25rem">{{ m.name }}</div>
          <div style="font-size: 0.8rem; color: #888; margin-top: 0.25rem">{{ m.description }}</div>
        </div>
      </div>
    </div>

    <!-- Scheduler Section -->
    <div class="card">
      <h3>Scheduler Jobs</h3>

      <p v-if="schedulerLoading" class="loading">Loading jobs...</p>
      <p v-if="schedulerError" class="error-msg">{{ schedulerError }}</p>
      <p v-if="schedulerSuccess" style="color: #065f46; font-size: 0.85rem; margin-bottom: 0.5rem">{{ schedulerSuccess }}</p>

      <div v-if="!schedulerLoading">
        <table>
          <thead>
            <tr>
              <th>Job Name</th>
              <th>Cron Expression</th>
              <th>Last Run</th>
              <th>Next Run</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="jobs.length === 0">
              <td colspan="5" style="text-align: center; color: #888; padding: 2rem">
                No scheduler jobs configured.
              </td>
            </tr>
            <tr v-for="job in jobs" :key="job.name">
              <td><strong>{{ job.name }}</strong></td>
              <td>
                <input
                  v-model="editedCrons[job.name]"
                  class="cron-input"
                  placeholder="*/5 * * * *"
                />
              </td>
              <td style="font-size: 0.85rem; color: #666">{{ formatTime(job.last_run) }}</td>
              <td style="font-size: 0.85rem; color: #666">{{ formatTime(job.next_run) }}</td>
              <td>
                <div style="display: flex; gap: 0.5rem">
                  <button
                    class="btn btn-secondary btn-sm"
                    :disabled="triggeringJob === job.name"
                    @click="handleTrigger(job.name)"
                  >
                    {{ triggeringJob === job.name ? 'Running...' : 'Run Now' }}
                  </button>
                  <button
                    v-if="cronChanged(job.name)"
                    class="btn btn-primary btn-sm"
                    :disabled="savingJob === job.name"
                    @click="handleSaveJob(job.name)"
                  >
                    {{ savingJob === job.name ? 'Saving...' : 'Save' }}
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mentor-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1rem;
  margin-top: 1rem;
}
.mentor-card {
  padding: 1rem;
  border: 2px solid #e5e7eb;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
}
.mentor-card:hover {
  border-color: #6366f1;
}
.mentor-card.selected {
  border-color: #6366f1;
  background: rgba(99, 102, 241, 0.05);
}
.mentor-card.disabled {
  opacity: 0.6;
  pointer-events: none;
}
.cron-input {
  width: 140px;
  padding: 0.25rem 0.5rem;
  border: 1px solid #ddd;
  border-radius: 4px;
  font-size: 0.85rem;
  font-family: monospace;
}
.cron-input:focus {
  border-color: #6366f1;
  outline: none;
}
.btn-sm {
  padding: 0.35rem 0.75rem;
  font-size: 0.8rem;
}
</style>
