import { ref, shallowRef, watch } from 'vue'
import { api } from './api'

export function usePaged(path, { pageSize = 12, keepPrevious = false } = {}) {
  const page = ref(1)
  const revision = ref(0)
  const data = shallowRef(null)
  const error = shallowRef(null)
  const loading = ref(true)
  watch([page, revision], async ([current], _, onCleanup) => {
    const controller = new AbortController()
    onCleanup(() => controller.abort())
    loading.value = true
    error.value = null
    if (!keepPrevious) data.value = null
    try {
      const result = await api(`${path}?page=${current}&page_size=${pageSize}`, { signal: controller.signal })
      if (!controller.signal.aborted) data.value = result
    } catch (cause) {
      if (!controller.signal.aborted) {
        data.value = null
        error.value = cause
      }
    } finally {
      if (!controller.signal.aborted) loading.value = false
    }
  }, { immediate: true })

  function reload(firstPage = false) {
    if (firstPage) page.value = 1
    revision.value += 1
  }
  return { page, data, error, loading, reload }
}
