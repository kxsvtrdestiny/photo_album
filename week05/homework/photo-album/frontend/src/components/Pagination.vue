<script setup>
import { computed } from 'vue'

const props = defineProps({ page: Number, pageSize: Number, total: Number, pending: Boolean })
const emit = defineEmits(['change'])
const last = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))
</script>

<template>
  <nav v-if="last > 1" class="pager" aria-label="分页">
    <button class="secondary" :disabled="pending || page <= 1" @click="emit('change', page - 1)">上一页</button>
    <span>第 {{ page }} / {{ last }} 页</span>
    <button class="secondary" :disabled="pending || page >= last" @click="emit('change', page + 1)">下一页</button>
  </nav>
</template>
