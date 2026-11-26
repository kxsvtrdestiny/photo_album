<script setup>
import { ref } from 'vue'
import { api } from '../api'
import Pagination from '../components/Pagination.vue'
import { usePaged } from '../paged'

const props = defineProps({ mine: Boolean })
const { page, data, error, loading, reload } = usePaged(props.mine ? '/albums' : '/albums/public')
const busy = ref(false)
const formError = ref('')
const feedback = ref('')

const deleting = ref(null)
const deleteError = ref('')
const deleteFeedback = ref('')

async function deleteAlbum(album) {
  if (!window.confirm(`确定删除「${album.name}」及其中的全部照片吗？此操作无法撤销。`)) return
  deleting.value = album.id
  deleteError.value = ''
  deleteFeedback.value = ''
  try {
    const result = await api(`/albums/${album.id}`, { method: 'DELETE' })
    deleteFeedback.value = result.cleanup_pending
      ? '相册已删除，但部分照片文件未能清理。'
      : '相册及其照片已删除。'
    reload(true)
  } catch (cause) {
    deleteError.value = cause.message
  } finally {
    deleting.value = null
  }
}

async function createAlbum(event) {
  const element = event.currentTarget
  const form = new FormData(element)
  busy.value = true
  formError.value = ''
  feedback.value = ''
  try {
    await api('/albums', {
      method: 'POST',
      body: {
        name: form.get('name').trim(),
        description: form.get('description'),
        is_public: form.get('visibility') === 'public',
      },
    })
    element.reset()
    feedback.value = '相册已创建。'
    reload(true)
  } catch (cause) {
    formError.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <h1>{{ mine ? '我的相册' : '公开广场' }}</h1>
  <details v-if="mine" class="panel create-panel" open>
    <summary>创建相册</summary>
    <form @submit.prevent="createAlbum">
      <div class="form-row">
        <label>相册名称 <input name="name" required maxlength="120"></label>
        <label>可见性
          <select name="visibility">
            <option value="private">私有</option>
            <option value="public">公开</option>
          </select>
        </label>
      </div>
      <label>描述 <textarea name="description" rows="2" maxlength="2000"></textarea></label>
      <p v-if="formError" class="notice error" role="alert">{{ formError }}</p>
      <div class="actions">
        <button :disabled="busy">{{ busy ? '创建中…' : '创建相册' }}</button>
        <span v-if="feedback" role="status">{{ feedback }}</span>
      </div>
    </form>
  </details>

  <p v-if="deleteError" class="notice error" role="alert">{{ deleteError }}</p>
  <p v-if="deleteFeedback" class="notice" role="status">{{ deleteFeedback }}</p>
  <p v-if="loading" class="notice" role="status">正在加载相册…</p>
  <div v-else-if="error" class="notice error" role="alert">
    {{ error.message }} <button class="secondary" @click="reload()">重试</button>
  </div>
  <template v-else-if="data">
    <p v-if="!data.items.length" class="notice">{{ mine ? '你还没有相册。' : '广场上还没有公开相册。' }}</p>
    <ul v-else class="album-grid" aria-label="相册列表">
      <li v-for="album in data.items" :key="album.id" class="panel album-card">
        <div class="card-head">
          <h2><RouterLink :to="{ name: 'photos', params: { id: album.id } }">{{ album.name }}</RouterLink></h2>
          <span v-if="mine" class="visibility" :class="{ public: album.is_public }">{{ album.is_public ? '公开' : '私有' }}</span>
        </div>
        <p v-if="album.description" class="description">{{ album.description }}</p>
        <div v-if="mine" class="actions">
          <button type="button" class="secondary danger" :aria-label="`删除相册 ${album.name}`" :disabled="deleting !== null" @click="deleteAlbum(album)">
            {{ deleting === album.id ? '删除中…' : '删除相册' }}
          </button>
        </div>
        <div v-if="album.creator" class="creator">
          <img v-if="album.creator.avatar_url" class="avatar" :src="album.creator.avatar_url" alt="">
          <span>{{ album.creator.username }}</span>
        </div>
      </li>
    </ul>
    <Pagination :page="data.page" :page-size="data.page_size" :total="data.total" @change="page = $event" />
  </template>
</template>
