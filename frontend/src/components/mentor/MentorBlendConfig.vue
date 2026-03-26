<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NSwitch, NSelect, NSlider, NSpace, NText } from 'naive-ui'
import type { MentorWithDomain, BlendConfig } from '@/types'

const props = defineProps<{
  mentors: MentorWithDomain[]
  currentMentorId: string
  blend: BlendConfig | null
}>()

const emit = defineEmits<{
  'update:blend': [blend: BlendConfig | null]
}>()

const isBlendOn = computed(() => !!props.blend)

const secondaryOptions = computed(() =>
  props.mentors
    .filter((m) => m.id !== props.currentMentorId)
    .map((m) => ({ label: m.name_en || m.name, value: m.id })),
)

const currentPrimaryName = computed(() => {
  const m = props.mentors.find((m) => m.id === props.currentMentorId)
  return m?.name_en || m?.name || props.currentMentorId
})

const secondaryName = computed(() => {
  if (!props.blend) return ''
  const m = props.mentors.find((m) => m.id === props.blend!.secondary_id)
  return m?.name_en || m?.name || props.blend!.secondary_id
})

function handleToggle(on: boolean) {
  if (on) {
    const firstOther = props.mentors.find((m) => m.id !== props.currentMentorId)
    if (firstOther) {
      emit('update:blend', { primary_id: props.currentMentorId, secondary_id: firstOther.id, weight: 70 })
    }
  } else {
    emit('update:blend', null)
  }
}

function handleSecondaryChange(val: string) {
  if (props.blend) {
    emit('update:blend', { ...props.blend, secondary_id: val })
  }
}

function handleWeightChange(val: number) {
  if (props.blend) {
    emit('update:blend', { ...props.blend, weight: val })
  }
}
</script>

<template>
  <NCard title="Blend Configuration" :bordered="false" style="box-shadow: 0 1px 3px rgba(0,0,0,0.08)">
    <NSpace vertical :size="16">
      <NSpace align="center" :size="12">
        <NText>Blend Mode</NText>
        <NSwitch :value="isBlendOn" @update:value="handleToggle" />
      </NSpace>

      <template v-if="blend">
        <NSpace vertical :size="12">
          <div>
            <NText depth="3" style="font-size: 13px">Secondary Mentor</NText>
            <NSelect
              :value="blend.secondary_id"
              :options="secondaryOptions"
              style="margin-top: 4px"
              @update:value="handleSecondaryChange"
            />
          </div>
          <div>
            <NText depth="3" style="font-size: 13px">Primary Weight</NText>
            <NSlider
              :value="blend.weight"
              :min="10"
              :max="90"
              :step="5"
              :format-tooltip="(v: number) => `${v}%`"
              style="margin-top: 4px"
              @update:value="handleWeightChange"
            />
          </div>
          <div style="padding: 12px; background: #f5f7fa; border-radius: 6px; font-size: 13px; color: #555">
            Primary: <strong>{{ currentPrimaryName }}</strong> ({{ blend.weight }}%)
            + Secondary: <strong>{{ secondaryName }}</strong> ({{ 100 - blend.weight }}%)
          </div>
        </NSpace>
      </template>
    </NSpace>
  </NCard>
</template>
