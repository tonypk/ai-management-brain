<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NSpin, NSpace, NTag, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import MentorGrid from '@/components/mentor/MentorGrid.vue'
import MentorBlendConfig from '@/components/mentor/MentorBlendConfig.vue'
import { listMentors, getMentorConfig, switchMentor } from '@/api'
import type { MentorWithDomain, BlendConfig } from '@/types'

const message = useMessage()

const loading = ref(true)
const switching = ref(false)
const mentors = ref<MentorWithDomain[]>([])
const currentMentorId = ref('')
const currentMentorName = ref('')
const blend = ref<BlendConfig | null>(null)

async function fetchData() {
  loading.value = true
  try {
    const [mentorList, config] = await Promise.all([listMentors(), getMentorConfig()])
    mentors.value = mentorList
    currentMentorId.value = config.current_mentor_id
    blend.value = config.current_blend
    const m = mentorList.find((m) => m.id === config.current_mentor_id)
    currentMentorName.value = m?.name_en || m?.name || config.current_mentor_id
  } catch (err: unknown) {
    message.error(`Failed to load mentors: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    loading.value = false
  }
}

async function handleSwitch(mentorId: string) {
  switching.value = true
  try {
    await switchMentor(mentorId)
    const m = mentors.value.find((m) => m.id === mentorId)
    currentMentorId.value = mentorId
    currentMentorName.value = m?.name_en || m?.name || mentorId
    message.success(`Switched to ${currentMentorName.value}`)
  } catch (err: unknown) {
    message.error(`Failed to switch: ${err instanceof Error ? err.message : 'Unknown error'}`)
  } finally {
    switching.value = false
  }
}

function handleBlendUpdate(val: BlendConfig | null) {
  blend.value = val
}

onMounted(fetchData)
</script>

<template>
  <div>
    <PageHeader title="Mentor Philosophy">
      <template #actions>
        <NSpace v-if="currentMentorName" align="center">
          <NTag type="success" size="medium" round>
            Active: {{ currentMentorName }}
          </NTag>
        </NSpace>
      </template>
    </PageHeader>

    <NSpin :show="loading">
      <NSpace vertical :size="24">
        <MentorGrid
          :mentors="mentors"
          :current-mentor-id="currentMentorId"
          :switching="switching"
          @switch="handleSwitch"
        />
        <MentorBlendConfig
          :mentors="mentors"
          :current-mentor-id="currentMentorId"
          :blend="blend"
          @update:blend="handleBlendUpdate"
        />
      </NSpace>
    </NSpin>
  </div>
</template>
