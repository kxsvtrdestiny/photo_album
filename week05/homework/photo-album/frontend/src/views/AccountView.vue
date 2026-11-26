<script setup>
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { api } from '../api'
import { setSession } from '../session'

const props = defineProps({ register: Boolean })
const router = useRouter()
const route = useRoute()
const busy = ref(false)
const error = ref('')
const redirect = typeof route.query.redirect === 'string' &&
  /^\/(mine|albums\/[^/?#]+)(?:[?#]|$)/.test(route.query.redirect)
  ? route.query.redirect : '/mine'

async function submit(event) {
  const form = new FormData(event.currentTarget)
  busy.value = true
  error.value = ''
  try {
    if (props.register) {
      const user = await api('/register', { method: 'POST', body: form, auth: false })
      await router.replace({ name: 'login', query: { username: user.username, registered: '1', redirect } })
    } else {
      const account = await api('/login', {
        method: 'POST',
        body: { username: form.get('username'), password: form.get('password') },
        auth: false,
      })
      setSession(account)
      await router.replace(redirect)
    }
  } catch (cause) {
    error.value = !props.register && cause.status === 401 ? '用户名或密码错误。' : cause.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <section class="panel auth-panel">
    <h1>{{ register ? '注册账号' : '登录' }}</h1>
    <p v-if="route.query.registered === '1'" class="notice" role="status">注册成功，请登录。</p>
    <form @submit.prevent="submit">
      <label>用户名
        <input name="username" :value="route.query.username ?? ''" autocomplete="username" required maxlength="64">
      </label>
      <label>密码
        <input name="password" type="password" :autocomplete="register ? 'new-password' : 'current-password'" required>
      </label>
      <label v-if="register">头像
        <input name="avatar" type="file" accept="image/jpeg,image/png,image/webp" required>
        <small>JPEG、PNG 或 WebP，最大 5 MiB。</small>
      </label>
      <p v-if="error" class="notice error" role="alert">{{ error }}</p>
      <button :disabled="busy">{{ busy ? '提交中…' : register ? '注册' : '登录' }}</button>
    </form>
    <p class="account-switch">
      {{ register ? '已有账号？' : '还没有账号？' }}
      <RouterLink :to="{ name: register ? 'login' : 'register', query: { redirect } }">{{ register ? '去登录' : '注册账号' }}</RouterLink>
    </p>
  </section>
</template>
