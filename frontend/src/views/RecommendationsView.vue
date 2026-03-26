<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { NSpin, NTabs, NTabPane, NEmpty, useMessage } from 'naive-ui'
import PageHeader from '@/components/shared/PageHeader.vue'
import RecommendationCard from '@/components/recommendations/RecommendationCard.vue'
import { listRecommendations } from '@/api/recommendations'
import type { Recommendation } from '@/types'

const message = useMessage()
const loading = ref(true)
const activeTab = ref('pending')
const pending = ref<Recommendation[]>([])
const executed = ref<Recommendation[]>([])
const dismissed = ref<Recommendation[]>([])

async function loadData() {
  loading.value = true
  try {
    const [p, e, d] = await Promise.all([
      listRecommendations('pending'),
      listRecommendations('executed'),
      listRecommendations('dismissed'),
    ])
    pending.value = p
    executed.value = e
    dismissed.value = d
  } catch {
    message.error('Failed to load recommendations')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
</script>

<template>
  <div>
    <PageHeader title="AI Recommendations" />

    <NSpin :show="loading">
      <NTabs v-model:value="activeTab" type="line">
        <NTabPane name="pending" :tab="`Pending (${pending.length})`">
          <NEmpty v-if="pending.length === 0" description="No pending recommendations" />
          <RecommendationCard
            v-for="rec in pending"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
        <NTabPane name="executed" :tab="`Executed (${executed.length})`">
          <NEmpty v-if="executed.length === 0" description="No executed recommendations" />
          <RecommendationCard
            v-for="rec in executed"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
        <NTabPane name="dismissed" :tab="`Dismissed (${dismissed.length})`">
          <NEmpty v-if="dismissed.length === 0" description="No dismissed recommendations" />
          <RecommendationCard
            v-for="rec in dismissed"
            :key="rec.id"
            :recommendation="rec"
            @refresh="loadData"
          />
        </NTabPane>
      </NTabs>
    </NSpin>
  </div>
</template>
