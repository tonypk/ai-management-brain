import { ref, watch, type Ref } from 'vue'

export function useLocalStorage<T>(key: string, defaultValue: T): Ref<T> {
  let initial = defaultValue
  try {
    const raw = localStorage.getItem(key)
    if (raw !== null) {
      initial = JSON.parse(raw) as T
    }
  } catch {
    // corrupt data — fall back to default
  }

  const data = ref(initial) as Ref<T>

  watch(data, (val) => {
    try {
      localStorage.setItem(key, JSON.stringify(val))
    } catch {
      // quota exceeded — silently ignore
    }
  }, { deep: true })

  return data
}
