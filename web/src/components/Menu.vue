<template>
  <el-menu :default-active="modelValue" class="side-menu" @select="$emit('update:modelValue', $event)">
    <div class="menu-label">工作负载</div>
    <el-menu-item v-for="item in menu" :key="item.key" :index="item.key">
      <span class="menu-icon">{{ item.icon }}</span>
      <span>{{ item.label }}</span>
    </el-menu-item>
  </el-menu>
</template>

<script setup lang="ts">
defineProps<{ modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()

const menu = [
  { key: 'apps', icon: '⬡', label: '应用' },
  { key: 'databases', icon: '▣', label: '数据库' },
  { key: 'redis', icon: '◇', label: 'Redis' },
  { key: 'repositories', icon: '⑂', label: '代码仓库' },
]
</script>
