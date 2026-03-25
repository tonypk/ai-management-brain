<script setup lang="ts">
import { NCard, NDescriptions, NDescriptionsItem } from 'naive-ui'
</script>

<template>
  <NCard title="Alert Rules" :bordered="false">
    <NDescriptions bordered :column="1" label-placement="left">
      <NDescriptionsItem label="Consecutive Misses Threshold">
        3 days — triggers warning; 5 days — triggers critical
      </NDescriptionsItem>
      <NDescriptionsItem label="Sentiment Drop Detection">
        Alerts when employee sentiment drops from positive to negative within 3 reports
      </NDescriptionsItem>
      <NDescriptionsItem label="Urgent Keywords">
        "blocked", "stuck", "help", "urgent", "critical" in check-in answers
      </NDescriptionsItem>
      <NDescriptionsItem label="Chase Frequency">
        Up to 2 reminders per day at 2-hour intervals
      </NDescriptionsItem>
    </NDescriptions>
  </NCard>
</template>
