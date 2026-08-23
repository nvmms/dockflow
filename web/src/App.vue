<template>
  <router-view v-if="route.meta.public" />
  <el-container v-else class="app-shell">
    <el-header class="topbar">
      <div class="brand"><span class="brand-mark">D</span><span>DockFlow</span></div>
      <div class="namespace-switcher">
        <span class="context-label">命名空间</span>
        <el-select v-model="currentNamespace" placeholder="选择命名空间" :loading="loadingNamespaces" @change="loadNamespaceDetail">
          <el-option v-for="item in namespaces" :key="item.name" :label="item.name" :value="item.name" />
        </el-select>
        <el-button text title="刷新" @click="loadNamespaces">↻</el-button>
        <el-button type="primary" plain @click="namespaceDialog = true">新建</el-button>
      </div>
      <div class="topbar-status"><span class="status-dot"></span><span>{{ auth.username }}</span><el-button text @click="logout">退出登录</el-button></div>
    </el-header>
    <el-container class="body-shell">
      <el-aside width="216px" class="sidebar">
        <Menu :model-value="activeView" @update:model-value="navigateMenu" />
        <div class="sidebar-footer">
          <div class="version">DockFlow Console</div>
          <a href="/api/v1/openapi.json" target="_blank">API 文档 ↗</a>
        </div>
      </el-aside>
      <el-main class="main-content">
        <div v-if="!currentNamespace" class="empty-state">
          <div class="empty-symbol">⌁</div>
          <h2>先创建一个命名空间</h2>
          <p>命名空间用于隔离应用、数据库和 Redis 资源。</p>
          <el-button type="primary" @click="namespaceDialog = true">创建命名空间</el-button>
        </div>
        <router-view v-else v-slot="{ Component }">
          <component :is="Component" :namespace="currentNamespace" @view-deployments="openDeployments" @back="backToApps" />
        </router-view>
      </el-main>
    </el-container>
  </el-container>

  <el-dialog v-model="namespaceDialog" title="创建命名空间" width="440px" destroy-on-close>
    <el-form label-position="top" @submit.prevent="createNamespace">
      <el-form-item label="名称" required>
        <el-input v-model="newNamespace" placeholder="例如 production" autofocus @keyup.enter="createNamespace" />
      </el-form-item>
      <div class="form-hint">将自动创建独立的 Docker 网络和可用子网。</div>
    </el-form>
    <template #footer><el-button @click="namespaceDialog = false">取消</el-button><el-button type="primary" :loading="creating" @click="createNamespace">创建</el-button></template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import { api, type Namespace } from './api'
import { auth } from './auth'
import Menu from './components/Menu.vue'

const route = useRoute()
const router = useRouter()
const activeView = computed(() => String(route.meta.menu || 'apps'))
const namespaces = ref<Namespace[]>([])
const currentNamespace = ref('')
const loadingNamespaces = ref(false)
const namespaceDialog = ref(false)
const newNamespace = ref('')
const creating = ref(false)

async function loadNamespaces() {
  loadingNamespaces.value = true
  try {
    namespaces.value = await api.get<Namespace[]>('/namespaces') || []
    const routeNamespace = typeof route.query.namespace === 'string' ? route.query.namespace : ''
    if (namespaces.value.some(n => n.name === routeNamespace)) currentNamespace.value = routeNamespace
    else if (!namespaces.value.some(n => n.name === currentNamespace.value)) currentNamespace.value = namespaces.value[0]?.name || ''
    syncNamespaceRoute()
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { loadingNamespaces.value = false }
}
function loadNamespaceDetail() {
  if (route.name === 'deployments') backToApps()
  else syncNamespaceRoute()
}
function syncNamespaceRoute() {
  const namespace = currentNamespace.value || undefined
  if (route.query.namespace !== namespace) router.replace({ query: { ...route.query, namespace } })
}
function navigateMenu(value: string) { router.push({ name: value, query: { namespace: currentNamespace.value || undefined } }) }
function openDeployments(app: string) { router.push({ name: 'deployments', params: { app }, query: { namespace: currentNamespace.value } }) }
function backToApps() { router.push({ name: 'apps', query: { namespace: currentNamespace.value } }) }
async function createNamespace() {
  if (!newNamespace.value.trim()) return ElMessage.warning('请输入命名空间名称')
  creating.value = true
  try {
    const created = await api.post<Namespace>('/namespaces', { name: newNamespace.value.trim() })
    namespaceDialog.value = false; newNamespace.value = ''; await loadNamespaces(); currentNamespace.value = created.name; syncNamespaceRoute()
    ElMessage.success('命名空间已创建')
  } catch (error) { ElMessage.error((error as Error).message) }
  finally { creating.value = false }
}
async function logout() {
  try { await api.post('/auth/logout') } finally { auth.username = ''; await router.replace('/login') }
}
function handleUnauthorized() { auth.username = ''; if (route.name !== 'login') router.replace({ name: 'login', query: { redirect: route.fullPath } }) }
onMounted(() => { window.addEventListener('dockflow:unauthorized', handleUnauthorized); if (!route.meta.public) loadNamespaces() })
onUnmounted(() => window.removeEventListener('dockflow:unauthorized', handleUnauthorized))
watch(() => route.meta.public, value => { if (!value && namespaces.value.length === 0) loadNamespaces() })
watch(() => route.query.namespace, value => {
  if (typeof value === 'string' && namespaces.value.some(item => item.name === value)) currentNamespace.value = value
})
</script>
