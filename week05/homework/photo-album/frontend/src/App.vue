<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from './api'
import { session, setSession } from './session'

const route = useRoute()
const router = useRouter()
const busy = ref(false)
const error = ref('')
const user = computed(() => session.value?.user)

watch(session, (value, previous) => {
  if (!value && previous && route.meta.protected && !busy.value) {
    router.replace({ name: 'login', query: { redirect: route.fullPath } })
  }
})

async function logout() {
  busy.value = true
  error.value = ''
  try {
    await api('/logout', { method: 'POST', auth: false })
    setSession(null)
    await router.replace({ name: 'public' })
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <header class="site-header">
    <div class="shell header-row">
      <RouterLink class="brand" :to="{ name: 'public' }">个人相册</RouterLink>
      <nav aria-label="主导航">
        <RouterLink :to="{ name: 'public' }">公开广场</RouterLink>
        <RouterLink :to="{ name: 'mine' }">我的相册</RouterLink>
      </nav>
      <div class="account">
        <template v-if="user">
          <img v-if="user.avatar_url" :src="user.avatar_url" alt="" class="avatar">
          <span class="username">{{ user.username }}</span>
          <button class="secondary" :disabled="busy" @click="logout">{{ busy ? '退出中…' : '退出登录' }}</button>
        </template>
        <template v-else>
          <RouterLink :to="{ name: 'login' }">登录</RouterLink>
          <RouterLink class="button" :to="{ name: 'register' }">注册</RouterLink>
        </template>
      </div>
    </div>
  </header>
  <main class="shell main">
    <p v-if="error" class="notice error" role="alert">{{ error }}</p>
    <RouterView :key="route.fullPath + ':' + (user?.id ?? 'guest')" />
  </main>
</template>
