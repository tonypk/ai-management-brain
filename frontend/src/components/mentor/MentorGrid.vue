<script setup lang="ts">
import { NGrid, NGridItem } from 'naive-ui'
import MentorCard from './MentorCard.vue'
import type { MentorWithDomain } from '@/types'

defineProps<{
  mentors: MentorWithDomain[]
  currentMentorId: string
  switching: boolean
}>()

const emit = defineEmits<{
  switch: [mentorId: string]
}>()
</script>

<template>
  <NGrid :cols="24" :x-gap="16" :y-gap="16" responsive="screen">
    <NGridItem
      v-for="mentor in mentors"
      :key="mentor.id"
      :span="24"
      :m="12"
      :l="8"
      :xl="6"
    >
      <MentorCard
        :mentor="mentor"
        :is-active="mentor.id === currentMentorId"
        :switching="switching"
        @switch="emit('switch', $event)"
      />
    </NGridItem>
  </NGrid>
</template>
