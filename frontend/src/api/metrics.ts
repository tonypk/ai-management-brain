import { get, post, put, del } from './client'
import type { Metric, MetricWithValue, MetricValue } from '@/types'

export async function listMetrics(): Promise<Metric[]> {
  const res = await get<{ data: Metric[] }>('/kpis')
  return res.data
}

export async function getMetric(id: string): Promise<Metric> {
  const res = await get<{ data: Metric }>(`/kpis/${id}`)
  return res.data
}

export async function createMetric(data: {
  name: string
  description?: string
  unit?: string
  frequency?: string
  target_value?: string
  alert_threshold?: string
  owner_id?: string
}): Promise<Metric> {
  const res = await post<{ data: Metric }>('/kpis', data)
  return res.data
}

export async function updateMetric(
  id: string,
  data: {
    name: string
    description?: string
    unit?: string
    frequency?: string
    target_value?: string
    alert_threshold?: string
    owner_id?: string
  },
): Promise<Metric> {
  const res = await put<{ data: Metric }>(`/kpis/${id}`, data)
  return res.data
}

export async function deleteMetric(id: string): Promise<void> {
  await del(`/kpis/${id}`)
}

export async function ingestMetricValue(data: {
  metric_id: string
  value: string
  recorded_at?: string
  source?: string
}): Promise<MetricValue> {
  const res = await post<{ data: MetricValue }>('/kpis/ingest', data)
  return res.data
}

export async function listMetricValues(metricId: string): Promise<MetricValue[]> {
  const res = await get<{ data: MetricValue[] }>(`/kpis/${metricId}/values`)
  return res.data
}

export async function getMetricsWithValues(): Promise<MetricWithValue[]> {
  const res = await get<{ data: MetricWithValue[] }>('/kpis/dashboard')
  return res.data
}
