<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { api } from '../api'
import Pagination from '../components/Pagination.vue'
import { usePaged } from '../paged'

const props = defineProps({ id: String })
const route = useRoute()
const path = `/albums/${encodeURIComponent(props.id)}/photos`
const { page, data, error, loading, reload } = usePaged(path, { keepPrevious: true })
const photoRange = computed(() => {
  if (!data.value.items.length) return `共 ${data.value.total} 张照片`
  const { page, page_size, total, items } = data.value
  const start = (page - 1) * page_size + 1
  const range = items.length === 1 ? start : `${start}–${start + items.length - 1}`
  return `显示第 ${range} 张，共 ${total} 张照片`
})
const busy = ref(false)
const uploadError = ref('')
const feedback = ref('')

async function upload(event) {
  const element = event.currentTarget
  const body = new FormData(element)
  busy.value = true
  uploadError.value = ''
  feedback.value = ''
  try {
    await api(path, { method: 'POST', body })
    element.reset()
    feedback.value = '照片已上传。'
    reload(true)
  } catch (cause) {
    uploadError.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <p class="back"><RouterLink :to="{ name: 'public' }">← 公开广场</RouterLink></p>
  <p v-if="loading && !data" class="notice" role="status">正在加载照片…</p>
  <section v-else-if="error" class="panel">
    <h1>{{ error.status === 403 ? '无权访问此私有相册' : '无法查看相册' }}</h1>
    <p class="notice error" role="alert">
      {{ error.status === 401 ? '这是私有相册，请使用相册主人的账号登录。' : error.message }}
    </p>
    <RouterLink v-if="error.status === 401" class="button" :to="{ name: 'login', query: { redirect: route.fullPath } }">去登录</RouterLink>
    <button v-else-if="![403, 404].includes(error.status)" class="secondary" @click="reload()">重试</button>
  </section>
  <template v-else-if="data">
    <div class="card-head">
      <h1>{{ data.album.name }}</h1>
      <span class="visibility" :class="{ public: data.album.is_public }">{{ data.album.is_public ? '公开' : '私有' }}</span>
    </div>
    <p v-if="data.album.description" class="description">{{ data.album.description }}</p>
    <form v-if="data.can_upload" class="panel upload-panel" @submit.prevent="upload">
      <label>选择照片
        <input name="photo" type="file" accept="image/jpeg,image/png,image/webp" required>
        <small>JPEG、PNG 或 WebP，最大 5 MiB。</small>
      </label>
      <p v-if="uploadError" class="notice error" role="alert">{{ uploadError }}</p>
      <div class="actions">
        <button :disabled="busy">{{ busy ? '上传中…' : '上传照片' }}</button>
        <span v-if="feedback" role="status">{{ feedback }}</span>
      </div>
    </form>
    <div v-if="data.total" class="photo-toolbar">
      <p role="status">{{ loading ? '正在加载照片…' : photoRange }}</p>
      <Pagination :page="data.page" :page-size="data.page_size" :total="data.total" :pending="loading" @change="page = $event" />
    </div>
    <p v-if="!data.items.length && !data.total" class="notice">相册里还没有照片。</p>
    <p v-else-if="!data.items.length" class="notice">这一页没有照片。</p>
    <ul v-else class="photo-grid" aria-label="照片列表" :aria-busy="loading">
      <li v-for="photo in data.items" :key="photo.id">
        <a :href="photo.file_path" target="_blank" rel="noopener">
          <img :src="photo.file_path" :alt="`照片 ${photo.id}`">
        </a>
      </li>
    </ul>
  </template>
</template>
