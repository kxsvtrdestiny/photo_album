<script setup>
import { onMounted, ref } from 'vue'

const loading = ref(false)
const status = ref('')

async function checkHealth() {
  loading.value = true
  status.value = '正在检查…'
  try {
    const response = await fetch('/api/v1/health')
    const result = await response.json()
    if (!response.ok || result.code !== 0 || result.data?.status !== 'ok') {
      throw new Error('Health check failed')
    }
    status.value = '后端和数据库正常'
  } catch {
    status.value = '无法确认服务状态，请检查后端是否已启动。'
  } finally {
    loading.value = false
  }
}

onMounted(checkHealth)
</script>

<template>
  <main>
    <h1>个人相册</h1>
    <p role="status">{{ status }}</p>
    <button type="button" :disabled="loading" @click="checkHealth">重新检查</button>
  </main>
</template>

<style>
body {
  margin: 0;
  background: #f5f7fa;
  color: #182438;
  font: 1rem/1.6 system-ui, sans-serif;
}

main {
  box-sizing: border-box;
  width: min(100% - 2rem, 38rem);
  margin: 12vh auto 0;
  padding: 2rem;
  background: white;
  border: 1px solid #dbe3ec;
  border-radius: 1rem;
}

h1 {
  margin: 0;
}

button {
  padding: 0.45rem 0.9rem;
  border: 1px solid #a5b3c6;
  border-radius: 0.5rem;
  background: white;
  color: inherit;
  font: inherit;
}
</style>
